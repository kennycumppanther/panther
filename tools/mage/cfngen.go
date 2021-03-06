package mage

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

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/panther-labs/panther/tools/dashboards"
)

// Generate CloudWatch dashboards as CloudFormation
func generateDashboards() error {
	dashboardResources := dashboards.Dashboards()
	logger.Debugf("deploy: cfngen: loaded %d dashboards", len(dashboardResources))

	template := map[string]interface{}{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              "Panther's CloudWatch monitoring dashboards",
	}

	resources := make(map[string]interface{}, len(dashboardResources))
	for _, dashboard := range dashboardResources {
		logicalID := strings.TrimPrefix(dashboard.Properties.DashboardName.Sub, "Panther")
		logicalID = strings.TrimSuffix(logicalID, "-${AWS::Region}")
		resources[logicalID] = dashboard
	}

	template["Resources"] = resources
	body, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("dashboard yaml marshal failed: %v", err)
	}

	body = append([]byte("# NOTE: template auto-generated by 'mage build:cfn', DO NOT EDIT\n"), body...)

	target := filepath.Join("deployments", "dashboards.yml")
	if err := writeFile(target, body); err != nil {
		return err
	}

	fmtLicense(target)
	return prettier(target)
}
