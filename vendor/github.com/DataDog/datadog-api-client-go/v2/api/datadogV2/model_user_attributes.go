// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// UserAttributes Attributes of user object returned by the API.
type UserAttributes struct {
	// Creation time of the user.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Whether the user is disabled.
	Disabled *bool `json:"disabled,omitempty"`
	// Email of the user.
	Email *string `json:"email,omitempty"`
	// Handle of the user.
	Handle *string `json:"handle,omitempty"`
	// URL of the user's icon.
	Icon *string `json:"icon,omitempty"`
	// Time that the user was last modified.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// Name of the user.
	Name datadog.NullableString `json:"name,omitempty"`
	// Whether the user is a service account.
	ServiceAccount *bool `json:"service_account,omitempty"`
	// Status of the user.
	Status *string `json:"status,omitempty"`
	// Title of the user.
	Title datadog.NullableString `json:"title,omitempty"`
	// Whether the user is verified.
	Verified *bool `json:"verified,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUserAttributes instantiates a new UserAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUserAttributes() *UserAttributes {
	this := UserAttributes{}
	return &this
}

// NewUserAttributesWithDefaults instantiates a new UserAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUserAttributesWithDefaults() *UserAttributes {
	this := UserAttributes{}
	return &this
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *UserAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *UserAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *UserAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetDisabled returns the Disabled field value if set, zero value otherwise.
func (o *UserAttributes) GetDisabled() bool {
	if o == nil || o.Disabled == nil {
		var ret bool
		return ret
	}
	return *o.Disabled
}

// GetDisabledOk returns a tuple with the Disabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetDisabledOk() (*bool, bool) {
	if o == nil || o.Disabled == nil {
		return nil, false
	}
	return o.Disabled, true
}

// HasDisabled returns a boolean if a field has been set.
func (o *UserAttributes) HasDisabled() bool {
	return o != nil && o.Disabled != nil
}

// SetDisabled gets a reference to the given bool and assigns it to the Disabled field.
func (o *UserAttributes) SetDisabled(v bool) {
	o.Disabled = &v
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *UserAttributes) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *UserAttributes) HasEmail() bool {
	return o != nil && o.Email != nil
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *UserAttributes) SetEmail(v string) {
	o.Email = &v
}

// GetHandle returns the Handle field value if set, zero value otherwise.
func (o *UserAttributes) GetHandle() string {
	if o == nil || o.Handle == nil {
		var ret string
		return ret
	}
	return *o.Handle
}

// GetHandleOk returns a tuple with the Handle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetHandleOk() (*string, bool) {
	if o == nil || o.Handle == nil {
		return nil, false
	}
	return o.Handle, true
}

// HasHandle returns a boolean if a field has been set.
func (o *UserAttributes) HasHandle() bool {
	return o != nil && o.Handle != nil
}

// SetHandle gets a reference to the given string and assigns it to the Handle field.
func (o *UserAttributes) SetHandle(v string) {
	o.Handle = &v
}

// GetIcon returns the Icon field value if set, zero value otherwise.
func (o *UserAttributes) GetIcon() string {
	if o == nil || o.Icon == nil {
		var ret string
		return ret
	}
	return *o.Icon
}

// GetIconOk returns a tuple with the Icon field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetIconOk() (*string, bool) {
	if o == nil || o.Icon == nil {
		return nil, false
	}
	return o.Icon, true
}

// HasIcon returns a boolean if a field has been set.
func (o *UserAttributes) HasIcon() bool {
	return o != nil && o.Icon != nil
}

// SetIcon gets a reference to the given string and assigns it to the Icon field.
func (o *UserAttributes) SetIcon(v string) {
	o.Icon = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *UserAttributes) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *UserAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *UserAttributes) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetName returns the Name field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UserAttributes) GetName() string {
	if o == nil || o.Name.Get() == nil {
		var ret string
		return ret
	}
	return *o.Name.Get()
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UserAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Name.Get(), o.Name.IsSet()
}

// HasName returns a boolean if a field has been set.
func (o *UserAttributes) HasName() bool {
	return o != nil && o.Name.IsSet()
}

// SetName gets a reference to the given datadog.NullableString and assigns it to the Name field.
func (o *UserAttributes) SetName(v string) {
	o.Name.Set(&v)
}

// SetNameNil sets the value for Name to be an explicit nil.
func (o *UserAttributes) SetNameNil() {
	o.Name.Set(nil)
}

// UnsetName ensures that no value is present for Name, not even an explicit nil.
func (o *UserAttributes) UnsetName() {
	o.Name.Unset()
}

// GetServiceAccount returns the ServiceAccount field value if set, zero value otherwise.
func (o *UserAttributes) GetServiceAccount() bool {
	if o == nil || o.ServiceAccount == nil {
		var ret bool
		return ret
	}
	return *o.ServiceAccount
}

// GetServiceAccountOk returns a tuple with the ServiceAccount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetServiceAccountOk() (*bool, bool) {
	if o == nil || o.ServiceAccount == nil {
		return nil, false
	}
	return o.ServiceAccount, true
}

// HasServiceAccount returns a boolean if a field has been set.
func (o *UserAttributes) HasServiceAccount() bool {
	return o != nil && o.ServiceAccount != nil
}

// SetServiceAccount gets a reference to the given bool and assigns it to the ServiceAccount field.
func (o *UserAttributes) SetServiceAccount(v bool) {
	o.ServiceAccount = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *UserAttributes) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *UserAttributes) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *UserAttributes) SetStatus(v string) {
	o.Status = &v
}

// GetTitle returns the Title field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UserAttributes) GetTitle() string {
	if o == nil || o.Title.Get() == nil {
		var ret string
		return ret
	}
	return *o.Title.Get()
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UserAttributes) GetTitleOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Title.Get(), o.Title.IsSet()
}

// HasTitle returns a boolean if a field has been set.
func (o *UserAttributes) HasTitle() bool {
	return o != nil && o.Title.IsSet()
}

// SetTitle gets a reference to the given datadog.NullableString and assigns it to the Title field.
func (o *UserAttributes) SetTitle(v string) {
	o.Title.Set(&v)
}

// SetTitleNil sets the value for Title to be an explicit nil.
func (o *UserAttributes) SetTitleNil() {
	o.Title.Set(nil)
}

// UnsetTitle ensures that no value is present for Title, not even an explicit nil.
func (o *UserAttributes) UnsetTitle() {
	o.Title.Unset()
}

// GetVerified returns the Verified field value if set, zero value otherwise.
func (o *UserAttributes) GetVerified() bool {
	if o == nil || o.Verified == nil {
		var ret bool
		return ret
	}
	return *o.Verified
}

// GetVerifiedOk returns a tuple with the Verified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UserAttributes) GetVerifiedOk() (*bool, bool) {
	if o == nil || o.Verified == nil {
		return nil, false
	}
	return o.Verified, true
}

// HasVerified returns a boolean if a field has been set.
func (o *UserAttributes) HasVerified() bool {
	return o != nil && o.Verified != nil
}

// SetVerified gets a reference to the given bool and assigns it to the Verified field.
func (o *UserAttributes) SetVerified(v bool) {
	o.Verified = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UserAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CreatedAt != nil {
		if o.CreatedAt.Nanosecond() == 0 {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Disabled != nil {
		toSerialize["disabled"] = o.Disabled
	}
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}
	if o.Handle != nil {
		toSerialize["handle"] = o.Handle
	}
	if o.Icon != nil {
		toSerialize["icon"] = o.Icon
	}
	if o.ModifiedAt != nil {
		if o.ModifiedAt.Nanosecond() == 0 {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Name.IsSet() {
		toSerialize["name"] = o.Name.Get()
	}
	if o.ServiceAccount != nil {
		toSerialize["service_account"] = o.ServiceAccount
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Title.IsSet() {
		toSerialize["title"] = o.Title.Get()
	}
	if o.Verified != nil {
		toSerialize["verified"] = o.Verified
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UserAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedAt      *time.Time             `json:"created_at,omitempty"`
		Disabled       *bool                  `json:"disabled,omitempty"`
		Email          *string                `json:"email,omitempty"`
		Handle         *string                `json:"handle,omitempty"`
		Icon           *string                `json:"icon,omitempty"`
		ModifiedAt     *time.Time             `json:"modified_at,omitempty"`
		Name           datadog.NullableString `json:"name,omitempty"`
		ServiceAccount *bool                  `json:"service_account,omitempty"`
		Status         *string                `json:"status,omitempty"`
		Title          datadog.NullableString `json:"title,omitempty"`
		Verified       *bool                  `json:"verified,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.CreatedAt = all.CreatedAt
	o.Disabled = all.Disabled
	o.Email = all.Email
	o.Handle = all.Handle
	o.Icon = all.Icon
	o.ModifiedAt = all.ModifiedAt
	o.Name = all.Name
	o.ServiceAccount = all.ServiceAccount
	o.Status = all.Status
	o.Title = all.Title
	o.Verified = all.Verified
	return nil
}
