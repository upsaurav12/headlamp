// Copyright 2025 The Kubernetes Authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kubernetes-sigs/headlamp/backend/pkg/cache"
	"github.com/kubernetes-sigs/headlamp/backend/pkg/kubeconfig"
)

var k8scache = cache.New[string]()

type responseCapture struct {
	http.ResponseWriter // The original ResponseWriter to which the captured response will eventually be written
	statusCode          int
	body                *bytes.Buffer // Stores the response
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	return r.ResponseWriter.Write(b)
}

// Initialize responseCapture with a http.ResponseWriter and empty bytes.Buffer for the body.
func Initialize(w http.ResponseWriter) *responseCapture {
	return &responseCapture{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

// The function extracts the namespace from the parameter from the given raw URL. This is used to make
// cache key more specific to a particular namespace.
func ExtractNamespace(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	namespace := parsedURL.Query().Get("namespace")

	return namespace, nil
}

// It is used to convert the captured response from gzip to string which can be easily convert
// []byte form for sending to client.
func GetResponseBody(bodyBytes []byte, encoding string) (string, error) {
	var dcmpBody []byte

	if encoding == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}

		decompressedBody, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("failed to decompress body: %w", err)
		}

		dcmpBody = decompressedBody

		reader.Close()
	} else {
		dcmpBody = bodyBytes
	}

	return string(dcmpBody), nil
}

// This function generates a unique cache key based on the requested URL's path (kind of resource),
// Namespace, and the kubernetes context. this ensures that the cached response are specific to the exact
// resource, namespace ,and cluster being requested.
func generateKey(url *url.URL, contextKey string) (string, error) {
	namespace, err := ExtractNamespace(url.String())
	if err != nil {
		return "", err
	}

	k := &cache.Key{
		Kind:      url.Path,
		Namespace: namespace,
		Cluster:   contextKey,
	}

	key, err := k.SHA()
	if err != nil {
		return "", err
	}

	return key, nil
}

// This function checks the user's permission to access the resource.
// If the user is authorized and has permission to view the resources, it returns true.
// Otherwise, it returns false if authorization fails.
func isAllowed(url *url.URL,
	kContext *kubeconfig.Context,
	w http.ResponseWriter,
	r *http.Request,
) bool {
	if strings.Contains(url.Path, "selfsubjectrulesreviews") {
		ssarResponse := Initialize(w)

		err := kContext.ProxyRequest(ssarResponse, r)
		if err != nil {
			return false
		}

		encoding := ssarResponse.Header().Get("Content-Encoding")
		bodyBytes := ssarResponse.body.Bytes()

		dcmp, err := GetResponseBody(bodyBytes, encoding)
		if err != nil {
			return false
		}

		if strings.Contains(dcmp, "Failure") {
			return false
		}
	}

	return true
}

// If the user has the permission to view the resources then it will check if the generated key is found
// in the cache if the key is present in the cache then it will return directly to the client in []byte form
// and returns true ,Otherwise it will return false.
func LoadfromCache(isAllowed bool, key string, w http.ResponseWriter) bool {
	if !isAllowed {
		return false
	}

	k8Resource, err := k8scache.Get(context.Background(), key)
	if err == nil && strings.TrimSpace(k8Resource) != "" { // No newline after "" before {
		w.Header().Set("Content-Type", "application/json")

		_, writeErr := w.Write([]byte(k8Resource))
		if writeErr == nil {
			return true
		}
	}

	return false
}

// If the key was not found inside the cache then this will make actual call to k8's
// and this will capture the response body and convert the captured response to string.
// After converting it will store the response with the key and TTL of 10*min.
func RequestToK8sAndStore(kContext *kubeconfig.Context,
	url *url.URL,
	w *responseCapture,
	r *http.Request,
	key string,
) error {
	if !strings.Contains(url.Path, "selfsubjectrulesreviews") {
		err := kContext.ProxyRequest(w, r)
		if err != nil {
			return err
		}
	}

	encoding := w.Header().Get("Content-Encoding")
	bodyBytes := w.body.Bytes()

	dcmpBody, err := GetResponseBody(bodyBytes, encoding)
	if err != nil {
		return err
	}

	if !strings.Contains(url.Path, "selfsubjectrulesreviews") {
		if err = k8scache.SetWithTTL(context.Background(), key, dcmpBody, 10*time.Minute); err != nil {
			return err
		}
	}

	return nil
}
