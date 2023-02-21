// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SyntheticsBasicAuth - Object to handle basic authentication when performing the test.
type SyntheticsBasicAuth struct {
	SyntheticsBasicAuthWeb         *SyntheticsBasicAuthWeb
	SyntheticsBasicAuthSigv4       *SyntheticsBasicAuthSigv4
	SyntheticsBasicAuthNTLM        *SyntheticsBasicAuthNTLM
	SyntheticsBasicAuthDigest      *SyntheticsBasicAuthDigest
	SyntheticsBasicAuthOauthClient *SyntheticsBasicAuthOauthClient
	SyntheticsBasicAuthOauthROP    *SyntheticsBasicAuthOauthROP

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// SyntheticsBasicAuthWebAsSyntheticsBasicAuth is a convenience function that returns SyntheticsBasicAuthWeb wrapped in SyntheticsBasicAuth.
func SyntheticsBasicAuthWebAsSyntheticsBasicAuth(v *SyntheticsBasicAuthWeb) SyntheticsBasicAuth {
	return SyntheticsBasicAuth{SyntheticsBasicAuthWeb: v}
}

// SyntheticsBasicAuthSigv4AsSyntheticsBasicAuth is a convenience function that returns SyntheticsBasicAuthSigv4 wrapped in SyntheticsBasicAuth.
func SyntheticsBasicAuthSigv4AsSyntheticsBasicAuth(v *SyntheticsBasicAuthSigv4) SyntheticsBasicAuth {
	return SyntheticsBasicAuth{SyntheticsBasicAuthSigv4: v}
}

// SyntheticsBasicAuthNTLMAsSyntheticsBasicAuth is a convenience function that returns SyntheticsBasicAuthNTLM wrapped in SyntheticsBasicAuth.
func SyntheticsBasicAuthNTLMAsSyntheticsBasicAuth(v *SyntheticsBasicAuthNTLM) SyntheticsBasicAuth {
	return SyntheticsBasicAuth{SyntheticsBasicAuthNTLM: v}
}

// SyntheticsBasicAuthDigestAsSyntheticsBasicAuth is a convenience function that returns SyntheticsBasicAuthDigest wrapped in SyntheticsBasicAuth.
func SyntheticsBasicAuthDigestAsSyntheticsBasicAuth(v *SyntheticsBasicAuthDigest) SyntheticsBasicAuth {
	return SyntheticsBasicAuth{SyntheticsBasicAuthDigest: v}
}

// SyntheticsBasicAuthOauthClientAsSyntheticsBasicAuth is a convenience function that returns SyntheticsBasicAuthOauthClient wrapped in SyntheticsBasicAuth.
func SyntheticsBasicAuthOauthClientAsSyntheticsBasicAuth(v *SyntheticsBasicAuthOauthClient) SyntheticsBasicAuth {
	return SyntheticsBasicAuth{SyntheticsBasicAuthOauthClient: v}
}

// SyntheticsBasicAuthOauthROPAsSyntheticsBasicAuth is a convenience function that returns SyntheticsBasicAuthOauthROP wrapped in SyntheticsBasicAuth.
func SyntheticsBasicAuthOauthROPAsSyntheticsBasicAuth(v *SyntheticsBasicAuthOauthROP) SyntheticsBasicAuth {
	return SyntheticsBasicAuth{SyntheticsBasicAuthOauthROP: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *SyntheticsBasicAuth) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into SyntheticsBasicAuthWeb
	err = json.Unmarshal(data, &obj.SyntheticsBasicAuthWeb)
	if err == nil {
		if obj.SyntheticsBasicAuthWeb != nil && obj.SyntheticsBasicAuthWeb.UnparsedObject == nil {
			jsonSyntheticsBasicAuthWeb, _ := json.Marshal(obj.SyntheticsBasicAuthWeb)
			if string(jsonSyntheticsBasicAuthWeb) == "{}" { // empty struct
				obj.SyntheticsBasicAuthWeb = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsBasicAuthWeb = nil
		}
	} else {
		obj.SyntheticsBasicAuthWeb = nil
	}

	// try to unmarshal data into SyntheticsBasicAuthSigv4
	err = json.Unmarshal(data, &obj.SyntheticsBasicAuthSigv4)
	if err == nil {
		if obj.SyntheticsBasicAuthSigv4 != nil && obj.SyntheticsBasicAuthSigv4.UnparsedObject == nil {
			jsonSyntheticsBasicAuthSigv4, _ := json.Marshal(obj.SyntheticsBasicAuthSigv4)
			if string(jsonSyntheticsBasicAuthSigv4) == "{}" { // empty struct
				obj.SyntheticsBasicAuthSigv4 = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsBasicAuthSigv4 = nil
		}
	} else {
		obj.SyntheticsBasicAuthSigv4 = nil
	}

	// try to unmarshal data into SyntheticsBasicAuthNTLM
	err = json.Unmarshal(data, &obj.SyntheticsBasicAuthNTLM)
	if err == nil {
		if obj.SyntheticsBasicAuthNTLM != nil && obj.SyntheticsBasicAuthNTLM.UnparsedObject == nil {
			jsonSyntheticsBasicAuthNTLM, _ := json.Marshal(obj.SyntheticsBasicAuthNTLM)
			if string(jsonSyntheticsBasicAuthNTLM) == "{}" { // empty struct
				obj.SyntheticsBasicAuthNTLM = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsBasicAuthNTLM = nil
		}
	} else {
		obj.SyntheticsBasicAuthNTLM = nil
	}

	// try to unmarshal data into SyntheticsBasicAuthDigest
	err = json.Unmarshal(data, &obj.SyntheticsBasicAuthDigest)
	if err == nil {
		if obj.SyntheticsBasicAuthDigest != nil && obj.SyntheticsBasicAuthDigest.UnparsedObject == nil {
			jsonSyntheticsBasicAuthDigest, _ := json.Marshal(obj.SyntheticsBasicAuthDigest)
			if string(jsonSyntheticsBasicAuthDigest) == "{}" { // empty struct
				obj.SyntheticsBasicAuthDigest = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsBasicAuthDigest = nil
		}
	} else {
		obj.SyntheticsBasicAuthDigest = nil
	}

	// try to unmarshal data into SyntheticsBasicAuthOauthClient
	err = json.Unmarshal(data, &obj.SyntheticsBasicAuthOauthClient)
	if err == nil {
		if obj.SyntheticsBasicAuthOauthClient != nil && obj.SyntheticsBasicAuthOauthClient.UnparsedObject == nil {
			jsonSyntheticsBasicAuthOauthClient, _ := json.Marshal(obj.SyntheticsBasicAuthOauthClient)
			if string(jsonSyntheticsBasicAuthOauthClient) == "{}" { // empty struct
				obj.SyntheticsBasicAuthOauthClient = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsBasicAuthOauthClient = nil
		}
	} else {
		obj.SyntheticsBasicAuthOauthClient = nil
	}

	// try to unmarshal data into SyntheticsBasicAuthOauthROP
	err = json.Unmarshal(data, &obj.SyntheticsBasicAuthOauthROP)
	if err == nil {
		if obj.SyntheticsBasicAuthOauthROP != nil && obj.SyntheticsBasicAuthOauthROP.UnparsedObject == nil {
			jsonSyntheticsBasicAuthOauthROP, _ := json.Marshal(obj.SyntheticsBasicAuthOauthROP)
			if string(jsonSyntheticsBasicAuthOauthROP) == "{}" { // empty struct
				obj.SyntheticsBasicAuthOauthROP = nil
			} else {
				match++
			}
		} else {
			obj.SyntheticsBasicAuthOauthROP = nil
		}
	} else {
		obj.SyntheticsBasicAuthOauthROP = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.SyntheticsBasicAuthWeb = nil
		obj.SyntheticsBasicAuthSigv4 = nil
		obj.SyntheticsBasicAuthNTLM = nil
		obj.SyntheticsBasicAuthDigest = nil
		obj.SyntheticsBasicAuthOauthClient = nil
		obj.SyntheticsBasicAuthOauthROP = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj SyntheticsBasicAuth) MarshalJSON() ([]byte, error) {
	if obj.SyntheticsBasicAuthWeb != nil {
		return json.Marshal(&obj.SyntheticsBasicAuthWeb)
	}

	if obj.SyntheticsBasicAuthSigv4 != nil {
		return json.Marshal(&obj.SyntheticsBasicAuthSigv4)
	}

	if obj.SyntheticsBasicAuthNTLM != nil {
		return json.Marshal(&obj.SyntheticsBasicAuthNTLM)
	}

	if obj.SyntheticsBasicAuthDigest != nil {
		return json.Marshal(&obj.SyntheticsBasicAuthDigest)
	}

	if obj.SyntheticsBasicAuthOauthClient != nil {
		return json.Marshal(&obj.SyntheticsBasicAuthOauthClient)
	}

	if obj.SyntheticsBasicAuthOauthROP != nil {
		return json.Marshal(&obj.SyntheticsBasicAuthOauthROP)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *SyntheticsBasicAuth) GetActualInstance() interface{} {
	if obj.SyntheticsBasicAuthWeb != nil {
		return obj.SyntheticsBasicAuthWeb
	}

	if obj.SyntheticsBasicAuthSigv4 != nil {
		return obj.SyntheticsBasicAuthSigv4
	}

	if obj.SyntheticsBasicAuthNTLM != nil {
		return obj.SyntheticsBasicAuthNTLM
	}

	if obj.SyntheticsBasicAuthDigest != nil {
		return obj.SyntheticsBasicAuthDigest
	}

	if obj.SyntheticsBasicAuthOauthClient != nil {
		return obj.SyntheticsBasicAuthOauthClient
	}

	if obj.SyntheticsBasicAuthOauthROP != nil {
		return obj.SyntheticsBasicAuthOauthROP
	}

	// all schemas are nil
	return nil
}

// NullableSyntheticsBasicAuth handles when a null is used for SyntheticsBasicAuth.
type NullableSyntheticsBasicAuth struct {
	value *SyntheticsBasicAuth
	isSet bool
}

// Get returns the associated value.
func (v NullableSyntheticsBasicAuth) Get() *SyntheticsBasicAuth {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSyntheticsBasicAuth) Set(val *SyntheticsBasicAuth) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSyntheticsBasicAuth) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSyntheticsBasicAuth) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSyntheticsBasicAuth initializes the struct as if Set has been called.
func NewNullableSyntheticsBasicAuth(val *SyntheticsBasicAuth) *NullableSyntheticsBasicAuth {
	return &NullableSyntheticsBasicAuth{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSyntheticsBasicAuth) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSyntheticsBasicAuth) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
