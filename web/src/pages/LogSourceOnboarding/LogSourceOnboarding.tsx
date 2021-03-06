/**
 * Panther is a Cloud-Native SIEM for the Modern Security Team.
 * Copyright (C) 2020 Panther Labs Inc
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import React from 'react';
import { Box, Card, FadeIn } from 'pouncejs';
import urls from 'Source/urls';
import LogSourceCard from 'Components/LogSourceCard';
import s3Logo from 'Assets/s3-minimal-logo.svg';
import sqsLogo from 'Assets/sqs-minimal-logo.svg';
import gsuiteLogo from 'Assets/gsuite-minimal-logo.svg';
import eventBridgeLogo from 'Assets/eventbridge-minimal-logo.svg';
import oktaLogo from 'Assets/okta-minimal-logo.svg';
import withSEO from 'Hoc/withSEO';
import { WizardPanel } from 'Components/Wizard';

const logSourcesConfig = [
  {
    logo: s3Logo,
    title: 'Amazon S3',
    type: 'S3',
  },
  {
    logo: sqsLogo,
    title: 'Amazon SQS',
    type: 'SQS',
  },
  {
    logo: eventBridgeLogo,
    title: 'Amazon EventBridge',
    type: 'eventbridge',
    disabled: true,
  },
  {
    logo: oktaLogo,
    title: 'Okta',
    type: 'okta',
    disabled: true,
  },
  {
    logo: gsuiteLogo,
    title: 'G Suite',
    type: 'gsuite',
    disabled: true,
  },
];
const LogSourceOnboarding: React.FC = () => {
  return (
    <FadeIn>
      <Card p={6} mb={6}>
        <WizardPanel>
          <WizardPanel.Heading
            title="Select a Source Type"
            subtitle="Please select the source type you want to configure from the list below"
          />
          <Box mx={10}>
            {logSourcesConfig.map(config => (
              <LogSourceCard
                key={config.title}
                logo={config.logo}
                title={config.title}
                disabled={config.disabled}
                to={`${urls.logAnalysis.sources.create(config.type)}`}
              />
            ))}
          </Box>
        </WizardPanel>
      </Card>
    </FadeIn>
  );
};

export default withSEO({ title: 'Create Log Source' })(LogSourceOnboarding);
