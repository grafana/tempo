// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentAttachmentPostmortemAttributes The attributes object for a postmortem attachment.
type IncidentAttachmentPostmortemAttributes struct {
	// The postmortem attachment.
	Attachment IncidentAttachmentsPostmortemAttributesAttachmentObject `json:"attachment"`
	// The type of postmortem attachment attributes.
	AttachmentType IncidentAttachmentPostmortemAttachmentType `json:"attachment_type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentAttachmentPostmortemAttributes instantiates a new IncidentAttachmentPostmortemAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentAttachmentPostmortemAttributes(attachment IncidentAttachmentsPostmortemAttributesAttachmentObject, attachmentType IncidentAttachmentPostmortemAttachmentType) *IncidentAttachmentPostmortemAttributes {
	this := IncidentAttachmentPostmortemAttributes{}
	this.Attachment = attachment
	this.AttachmentType = attachmentType
	return &this
}

// NewIncidentAttachmentPostmortemAttributesWithDefaults instantiates a new IncidentAttachmentPostmortemAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentAttachmentPostmortemAttributesWithDefaults() *IncidentAttachmentPostmortemAttributes {
	this := IncidentAttachmentPostmortemAttributes{}
	var attachmentType IncidentAttachmentPostmortemAttachmentType = INCIDENTATTACHMENTPOSTMORTEMATTACHMENTTYPE_POSTMORTEM
	this.AttachmentType = attachmentType
	return &this
}

// GetAttachment returns the Attachment field value.
func (o *IncidentAttachmentPostmortemAttributes) GetAttachment() IncidentAttachmentsPostmortemAttributesAttachmentObject {
	if o == nil {
		var ret IncidentAttachmentsPostmortemAttributesAttachmentObject
		return ret
	}
	return o.Attachment
}

// GetAttachmentOk returns a tuple with the Attachment field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentPostmortemAttributes) GetAttachmentOk() (*IncidentAttachmentsPostmortemAttributesAttachmentObject, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attachment, true
}

// SetAttachment sets field value.
func (o *IncidentAttachmentPostmortemAttributes) SetAttachment(v IncidentAttachmentsPostmortemAttributesAttachmentObject) {
	o.Attachment = v
}

// GetAttachmentType returns the AttachmentType field value.
func (o *IncidentAttachmentPostmortemAttributes) GetAttachmentType() IncidentAttachmentPostmortemAttachmentType {
	if o == nil {
		var ret IncidentAttachmentPostmortemAttachmentType
		return ret
	}
	return o.AttachmentType
}

// GetAttachmentTypeOk returns a tuple with the AttachmentType field value
// and a boolean to check if the value has been set.
func (o *IncidentAttachmentPostmortemAttributes) GetAttachmentTypeOk() (*IncidentAttachmentPostmortemAttachmentType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AttachmentType, true
}

// SetAttachmentType sets field value.
func (o *IncidentAttachmentPostmortemAttributes) SetAttachmentType(v IncidentAttachmentPostmortemAttachmentType) {
	o.AttachmentType = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentAttachmentPostmortemAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attachment"] = o.Attachment
	toSerialize["attachment_type"] = o.AttachmentType

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentAttachmentPostmortemAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attachment     *IncidentAttachmentsPostmortemAttributesAttachmentObject `json:"attachment"`
		AttachmentType *IncidentAttachmentPostmortemAttachmentType              `json:"attachment_type"`
	}{}
	all := struct {
		Attachment     IncidentAttachmentsPostmortemAttributesAttachmentObject `json:"attachment"`
		AttachmentType IncidentAttachmentPostmortemAttachmentType              `json:"attachment_type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attachment == nil {
		return fmt.Errorf("required field attachment missing")
	}
	if required.AttachmentType == nil {
		return fmt.Errorf("required field attachment_type missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.AttachmentType; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attachment.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attachment = all.Attachment
	o.AttachmentType = all.AttachmentType
	return nil
}
