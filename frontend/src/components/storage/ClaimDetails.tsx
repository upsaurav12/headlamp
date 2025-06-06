/*
 * Copyright 2025 The Kubernetes Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import PersistentVolumeClaim from '../../lib/k8s/persistentVolumeClaim';
import Link from '../common/Link';
import { DetailsGrid } from '../common/Resource';
import { StatusLabelByPhase } from './utils';

export function makePVCStatusLabel(item: PersistentVolumeClaim) {
  const status = item.status!.phase;
  return StatusLabelByPhase(status!);
}

export default function VolumeClaimDetails(props: {
  name?: string;
  namespace?: string;
  cluster?: string;
}) {
  const params = useParams<{ namespace: string; name: string }>();
  const { name = params.name, namespace = params.namespace, cluster } = props;
  const { t } = useTranslation(['glossary', 'translation']);

  return (
    <DetailsGrid
      resourceType={PersistentVolumeClaim}
      name={name}
      namespace={namespace}
      cluster={cluster}
      withEvents
      extraInfo={item =>
        item && [
          {
            name: t('translation|Status'),
            value: makePVCStatusLabel(item),
          },
          {
            name: t('Capacity'),
            value: item.spec!.resources!.requests.storage,
          },
          {
            name: t('Access Modes'),
            value: item.spec!.accessModes!.join(', '),
          },
          {
            name: t('Volume Mode'),
            value: item.spec!.volumeMode,
          },
          {
            name: t('Storage Class'),
            value: (
              <Link
                routeName="storageClass"
                params={{ name: item.spec!.storageClassName }}
                activeCluster={item.cluster}
                tooltip
              >
                {item.spec!.storageClassName}
              </Link>
            ),
          },
        ]
      }
    />
  );
}
