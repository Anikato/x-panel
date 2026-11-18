package main

import (
	"encoding/json"

	"xpanel/app/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const appearanceApplyCapability = "appearance.apply"
const appearanceSettingKey = "ManagedAppearanceConfig"

func invokeApplyAppearance(req invokeRequest, db *gorm.DB) invokeResponse {
	reply := invokeResponse{Capability: appearanceApplyCapability}
	var payload struct {
		Profile *model.AppearanceProfile `json:"profile"`
		Clear   bool                     `json:"clear"`
	}
	if json.Unmarshal(req.Payload, &payload) != nil ||
		payload.Clear == (payload.Profile != nil) ||
		(payload.Profile != nil && payload.Profile.Validate() != nil) {
		reply.Error = "invalid_request"
		return reply
	}
	if db == nil {
		reply.Error = "database_unreadable"
		return reply
	}

	value := ""
	if payload.Profile != nil {
		buffer, err := json.Marshal(payload.Profile)
		if err != nil {
			reply.Error = "internal"
			return reply
		}
		value = string(buffer)
	}
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&model.Setting{Key: appearanceSettingKey, Value: value}).Error
	if err != nil {
		reply.Error = invokeDBErrorCode(classifyInvokeDBError(err))
		return reply
	}
	reply.OK = true
	return reply
}
