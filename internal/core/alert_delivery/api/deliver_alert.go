package api

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
	"go.uber.org/zap"

	alertmodels "github.com/panther-labs/panther/api/lambda/alerts/models"
	"github.com/panther-labs/panther/api/lambda/delivery/models"
	outputmodels "github.com/panther-labs/panther/api/lambda/outputs/models"
	delivery "github.com/panther-labs/panther/internal/core/alert_delivery/delivery"
	"github.com/panther-labs/panther/pkg/gatewayapi"
	"github.com/panther-labs/panther/pkg/genericapi"
)

// DeliverAlert sends a specific alert to the specified destinations.
func (API) DeliverAlert(input *models.DeliverAlertInput) (*models.DeliverAlertOutput, error) {
	// First, fetch the alert
	alertItem, err := alertsDB.GetAlert(&input.AlertID)
	if err != nil {
		zap.L().Error("Failed to fetch alert from ddb", zap.Error(err))
		return nil, err
	}

	// If the alertId was not found, log and return
	if alertItem == nil {
		zap.L().Error("Alert not found", zap.String("AlertID", input.AlertID))
		return nil, &genericapi.InternalError{
			Message: "Unable to find the specified alert!"}
	}

	// TODO: Fetch the Policy or Rule associated with the alert to fill in the gaps
	// ...
	//

	// TODO: Merge the related fields from the Policy/Rule from above into the alert.
	// For now, we are taking the data from Dynamo and populating as much as we have.
	// Eventually, sending an alert should be _exactly_ the same as if it were triggered
	// from a Policy or a Rule.
	alert := &models.Alert{
		AnalysisID: alertItem.AlertID,
		// Type:       alertItem.LogTypes,
		CreatedAt: alertItem.CreationTime,
		Severity:  alertItem.Severity,
		OutputIds: alertItem.OutputIds,
		// AnalysisDescription: alertItem.Title,
		AnalysisName: alertItem.RuleDisplayName,
		Version:      &alertItem.RuleVersion,
		Runbook:      alertItem.RuleDisplayName,
		// Tags:         alertItem.Tags,
		AlertID:    &alertItem.AlertID,
		Title:      alertItem.Title,
		RetryCount: 0,
	}

	// Fetch outputIds from ddb
	outputIds, err := delivery.GetOutputs()
	if err != nil {
		zap.L().Error("Failed to fetch outputIds", zap.Error(err))
		return nil, err
	}

	// Check to see if the input outputIds are valid
	invalidOutputIds, validOutputIds := validateInputs(input.OutputIds, outputIds)
	if len(invalidOutputIds) > 0 {
		zap.L().Error("Invalid outputIds specified", zap.Strings("OutputIds", invalidOutputIds))
		return nil, &genericapi.InternalError{
			Message: "Invalid destination(s) specified!"}
	}

	// Create our Alert -> Output mappings
	alertOutputMap := make(delivery.AlertOutputMap)
	// Create a slice to be universal with the SQS payload format
	alerts := []*models.Alert{alert}
	for _, alert := range alerts {
		alertOutputMap[alert] = validOutputIds
	}

	// Deliver alerts to the specified destination(s) and obtain each response status
	dispatchStatuses := delivery.DeliverAlerts(alertOutputMap)
	for _, delivery := range dispatchStatuses {
		if !delivery.Success {
			zap.L().Error(
				"failed to send alert to output",
				zap.String("alertID", delivery.AlertID),
				zap.String("outputID", delivery.OutputID),
				zap.String("message", delivery.Message),
			)
			return nil, &genericapi.InternalError{
				Message: "Failed to send the alert: " + delivery.Message}
		}
	}

	// TODO: Record the responses to ddb
	// ...
	//

	// Convert the alerts to summaries to update the frontend
	alertSummary := alertUtils.AlertItemToSummary(alertItem)

	// Since this API accepts only one alertID, we can directly
	// access the first item in the lists to add the delivery status
	alertSummary.DeliverySuccess = dispatchStatuses[0].Success
	alertSummary.DeliveryResponses = []*alertmodels.DeliveryResponse{
		{
			OutputID: dispatchStatuses[0].OutputID,
			Response: dispatchStatuses[0].Message,
			Success:  dispatchStatuses[0].Success,
		},
	}
	gatewayapi.ReplaceMapSliceNils(alertSummary)
	return alertSummary, nil
}

func validateInputs(inputIds []string, outputIds []*outputmodels.AlertOutput) (invalid []string, valid []*outputmodels.AlertOutput) {
	for i, outputID := range inputIds {
		if !contains(outputIds, outputID) {
			invalid = append(invalid, outputID)
		} else {
			valid = append(valid, outputIds[i])
		}
	}
	return invalid, valid
}

func contains(list []*outputmodels.AlertOutput, outputID string) bool {
	for _, item := range list {
		if *item.OutputID == outputID {
			return true
		}
	}
	return false
}

// updateDelivery - performs an API query to get a list of outputs
// func updateDelivery() ([]*outputmodels.AlertOutput, error) {
// 	zap.L().Debug("getting default outputs")
// 	input := outputmodels.LambdaInput{GetOutputsWithSecrets: &outputmodels.GetOutputsWithSecretsInput{}}
// 	var outputs outputmodels.GetOutputsOutput
// 	if err := genericapi.Invoke(lambdaClient, outputsAPI, &input, &outputs); err != nil {
// 		return nil, err
// 	}
// 	return outputs, nil
// }
