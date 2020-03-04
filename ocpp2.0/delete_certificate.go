package ocpp2

import (
	"gopkg.in/go-playground/validator.v9"
	"reflect"
)

// -------------------- Clear Display (CSMS -> CS) --------------------

// Status returned in response to DeleteCertificateRequest.
type DeleteCertificateStatus string

const (
	DeleteCertificateStatusAccepted DeleteCertificateStatus = "Accepted"
	DeleteCertificateStatusFailed   DeleteCertificateStatus = "Failed"
	DeleteCertificateStatusNotFound DeleteCertificateStatus = "NotFound"
)

func isValidDeleteCertificateStatus(fl validator.FieldLevel) bool {
	status := DeleteCertificateStatus(fl.Field().String())
	switch status {
	case DeleteCertificateStatusAccepted, DeleteCertificateStatusFailed, DeleteCertificateStatusNotFound:
		return true
	default:
		return false
	}
}

// The field definition of the DeleteCertificate request payload sent by the CSMS to the Charging Station.
type DeleteCertificateRequest struct {
	CertificateHashData CertificateHashData `json:"certificateHashData" validate:"required"`
}

// This field definition of the DeleteCertificate confirmation payload, sent by the Charging Station to the CSMS in response to a DeleteCertificateRequest.
// In case the request was invalid, or couldn't be processed, an error will be sent instead.
type DeleteCertificateConfirmation struct {
	Status DeleteCertificateStatus `json:"status" validate:"required,deleteCertificateStatus"`
}

// The CSMS requests the Charging Station to delete a specific installed certificate by sending a DeleteCertificateRequest.
// The Charging Station responds with a DeleteCertificateResponse.
type DeleteCertificateFeature struct{}

func (f DeleteCertificateFeature) GetFeatureName() string {
	return DeleteCertificateFeatureName
}

func (f DeleteCertificateFeature) GetRequestType() reflect.Type {
	return reflect.TypeOf(DeleteCertificateRequest{})
}

func (f DeleteCertificateFeature) GetConfirmationType() reflect.Type {
	return reflect.TypeOf(DeleteCertificateConfirmation{})
}

func (r DeleteCertificateRequest) GetFeatureName() string {
	return DeleteCertificateFeatureName
}

func (c DeleteCertificateConfirmation) GetFeatureName() string {
	return DeleteCertificateFeatureName
}

// Creates a new DeleteCertificateRequest, containing all required fields. There are no optional fields for this message.
func NewDeleteCertificateRequest(certificateHashData CertificateHashData) *DeleteCertificateRequest {
	return &DeleteCertificateRequest{CertificateHashData: certificateHashData}
}

// Creates a new DeleteCertificateConfirmation, containing all required fields. There are no optional fields for this message.
func NewDeleteCertificateConfirmation(status DeleteCertificateStatus) *DeleteCertificateConfirmation {
	return &DeleteCertificateConfirmation{Status: status}
}

func init() {
	_ = Validate.RegisterValidation("deleteCertificateStatus", isValidDeleteCertificateStatus)
}