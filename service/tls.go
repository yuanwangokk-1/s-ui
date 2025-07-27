package service

import (
	"encoding/json"
	"s-ui/database"
	"s-ui/database/model"
	"s-ui/util/common"

	"gorm.io/gorm"
)

type TlsService struct {
	InboundService
}

func (s *TlsService) GetAll() ([]model.Tls, error) {
	db := database.GetDB()
	tlsConfig := []model.Tls{}
	err := db.Model(model.Tls{}).Scan(&tlsConfig).Error
	if err != nil {
		return nil, err
	}

	return tlsConfig, nil
}

func (s *TlsService) Save(tx *gorm.DB, action string, data json.RawMessage) ([]uint, error) {
	var err error
	var inboundIds []uint

	switch action {
	case "new", "edit":
		var tls model.Tls
		err = json.Unmarshal(data, &tls)
		if err != nil {
			return nil, err
		}
		err = tx.Save(&tls).Error
		if err != nil {
			return nil, err
		}
		err = tx.Model(model.Inbound{}).Select("id").Where("tls_id = ?", tls.Id).Scan(&inboundIds).Error
		if err != nil {
			return nil, err
		}
		return inboundIds, nil
	case "del":
		var id uint
		err = json.Unmarshal(data, &id)
		if err != nil {
			return nil, err
		}
		var inboundCount int64
		err = tx.Model(model.Inbound{}).Where("tls_id = ?", id).Count(&inboundCount).Error
		if err != nil {
			return nil, err
		}
		if inboundCount > 0 {
			return nil, common.NewError("tls in use")
		}
		err = tx.Where("id = ?", id).Delete(model.Tls{}).Error
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
