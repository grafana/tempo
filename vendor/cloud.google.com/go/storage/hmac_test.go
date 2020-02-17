// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/internal/testutil"
	"google.golang.org/api/googleapi"
)

func TestHMACKeyHandle_GetParsing(t *testing.T) {
	mt := &mockTransport{}
	client := mockClient(t, mt)
	projectID := "hmackey-project-id"
	ctx := context.Background()

	tests := []struct {
		res     string
		want    *HMACKey
		wantErr string
	}{
		{
			res: fmt.Sprintf(`
                            {
                                "kind": "storage#hmacKeyMetadata",
                                "projectId":%q,"state":"ACTIVE",
                                "timeCreated": "2019-07-06T11:21:58+00:00",
                                "updated": "2019-07-06T11:22:18+00:00"
                            }`, projectID),
			want: &HMACKey{
				State:       Active,
				ProjectID:   projectID,
				UpdatedTime: time.Date(2019, 07, 06, 11, 22, 18, 0, time.UTC),
				CreatedTime: time.Date(2019, 07, 06, 11, 21, 58, 0, time.UTC),
			},
		},
		{
			res: fmt.Sprintf(`
                            {
                                "kind": "storage#hmacKeyMetadata",
                                "projectId":%q,"state":"ACTIVE",
                                "timeCreated": "2019-07-06T11:21:58+00:00",
                                "updated": "2019-07-06T11:22:18+00:00"
                            }`, projectID),
			want: &HMACKey{
				State:       Active,
				ProjectID:   projectID,
				UpdatedTime: time.Date(2019, 07, 06, 11, 22, 18, 0, time.UTC),
				CreatedTime: time.Date(2019, 07, 06, 11, 21, 58, 0, time.UTC),
			},
		},
		{
			res: `{}`,
			// CreatedTime must be formatted in RFC 3339.
			wantErr: `CreatedTime: parsing time "" as "2006-01-02T15:04:05Z07:00"`,
		},
		{
			res: `{"timeCreated": "2019-07-foo"}`,
			// CreatedTime must be formatted in RFC 3339.
			wantErr: `CreatedTime: parsing time "2019-07-foo" as "2006-01-02T15:04:05Z07:00"`,
		},
		{
			res: `{
                                "kind": "storage#hmacKeyMetadata",
                                "state":"INACTIVE",
                                "timeCreated": "2019-07-06T11:21:58+00:00"
                            }`,
			// UpdatedTime must be formatted in RFC 3339.
			wantErr: `UpdatedTime: parsing time "" as "2006-01-02T15:04:05Z07:00"`,
		},
	}

	for i, tt := range tests {
		mt.addResult(&http.Response{
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(tt.res)),
			Status:        "OK",
			StatusCode:    200,
			Body:          bodyReader(tt.res),
		}, nil)
		hkh := client.HMACKeyHandle(projectID, "some-access-key-id")
		got, err := hkh.Get(ctx)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d: failed to match errors:\ngot:  %q\nwant: %q", i, err, tt.wantErr)
			}
			if got != nil {
				t.Errorf("#%d: unexpectedly got a non-nil result: %#v\n", i, got)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: got an unexpected error: %v", i, err)
			continue
		}

		if diff := testutil.Diff(got, tt.want); diff != "" {
			t.Errorf("#%d: got - want +\n\n%s", i, diff)
		}
	}
}

func TestHMACKeyHandle_Get_NotFound(t *testing.T) {
	mt := &mockTransport{}
	client := mockClient(t, mt)
	ctx := context.Background()

	mt.addResult(&http.Response{
		ProtoMajor: 2,
		ProtoMinor: 0,
		Status:     "OK",
		StatusCode: http.StatusNotFound,
		Body:       bodyReader("Access ID not found in project"),
	}, nil)

	hkh := client.HMACKeyHandle("project-id", "some-access-key-id")
	_, gotErr := hkh.Get(ctx)

	wantErr := &googleapi.Error{
		Body:    "Access ID not found in project",
		Code:    http.StatusNotFound,
		Message: "",
	}
	if diff := testutil.Diff(gotErr, wantErr); diff != "" {
		t.Fatalf("Error mismatch, got - want +\n%s", diff)
	}
}

func TestHMACKeyHandle_Delete(t *testing.T) {
	mt := &mockTransport{}
	client := mockClient(t, mt)
	ctx := context.Background()

	tests := []struct {
		statusCode int
		msg        string
		wantErr    error
	}{
		{
			statusCode: http.StatusBadRequest,
			msg:        "Cannot delete keys in 'ACTIVE' state",
			wantErr: &googleapi.Error{
				Code: http.StatusBadRequest, Message: "Cannot delete keys in 'ACTIVE' state",
				Body: `{"error":{"message":"Cannot delete keys in 'ACTIVE' state"}}`,
			},
		},
		{
			statusCode: http.StatusNotFound,
			msg:        "random message",
			wantErr: &googleapi.Error{
				Code: http.StatusNotFound, Message: "random message",
				Body: `{"error":{"message":"random message"}}`,
			},
		},
		{
			statusCode: http.StatusNotFound,
			msg:        "Access ID not found in project",
			wantErr: &googleapi.Error{
				Code: http.StatusNotFound, Message: "Access ID not found in project",
				Body: `{"error":{"message":"Access ID not found in project"}}`,
			},
		},
	}

	for i, tt := range tests {
		mt.addResult(&http.Response{
			ProtoMajor: 2,
			ProtoMinor: 0,
			Status:     tt.msg,
			StatusCode: tt.statusCode,
			Body:       bodyReader(fmt.Sprintf(`{"error":{"message":%q}}`, tt.msg)),
		}, nil)

		hkh := client.HMACKeyHandle("project", "access-key-id")
		err := hkh.Delete(ctx)

		if diff := testutil.Diff(err, tt.wantErr); diff != "" {
			t.Errorf("#%d: error mismatch got - want +\n%s", i, diff)
		}
	}
}

func TestHMACKeyHandle_Create(t *testing.T) {
	mt := &mockTransport{}
	client := mockClient(t, mt)
	projectID := "hmackey-project-id"
	serviceAccountEmail := "service-account-email-1"
	ctx := context.Background()

	tests := []struct {
		res     string
		want    *HMACKey
		wantErr string
	}{
		{
			res: `
                            {
                                "kind": "storage#hmackey",
				"secret":"bGoa+V7g/yqDXvKRqq+JTFn4uQZbPiQJo4pf9RzJ",
                                "metadata": {
                                    "projectId":"project-id","state":"ACTIVE",
                                    "timeCreated": "2019-07-06T11:21:58+00:00",
                                    "updated": "2019-07-06T11:22:18+00:00"
                                }
                            }`,
			want: &HMACKey{
				Secret:      "bGoa+V7g/yqDXvKRqq+JTFn4uQZbPiQJo4pf9RzJ",
				State:       Active,
				ProjectID:   "project-id",
				UpdatedTime: time.Date(2019, 07, 06, 11, 22, 18, 0, time.UTC),
				CreatedTime: time.Date(2019, 07, 06, 11, 21, 58, 0, time.UTC),
			},
		},
		{
			res:     `{}`,
			wantErr: "Metadata cannot be nil",
		},
		{
			res: `{"metadata":{}}`,
			// CreatedTime must be non-empty and it must formatted in RFC 3339.
			wantErr: `CreatedTime: parsing time "" as "2006-01-02T15:04:05Z07:00"`,
		},
		{
			res: `{"metadata":{"timeCreated": "2019-07-foo"}}`,
			// CreatedTime must be formatted in RFC 3339.
			wantErr: `CreatedTime: parsing time "2019-07-foo" as "2006-01-02T15:04:05Z07:00"`,
		},
		{
			res: `{
                                "kind": "storage#hmackey",
				"secret":"bGoa+V7g/yqDXvKRqq+JTFn4uQZbPiQJo4pf9RzJ",
                                "metadata":{
                                    "kind": "storage#hmacKeyMetadata",
                                    "state":"ACTIVE",
                                    "timeCreated": "2019-07-06T12:11:33+00:00",
                                    "projectId": "project-id",
                                    "updated": ""
                                }
                            }`,
			// ONLY during creation is it okay for UpdatedTime to not be set.
			want: &HMACKey{
				Secret:      "bGoa+V7g/yqDXvKRqq+JTFn4uQZbPiQJo4pf9RzJ",
				State:       Active,
				ProjectID:   "project-id",
				CreatedTime: time.Date(2019, 07, 06, 12, 11, 33, 0, time.UTC),
			},
		},
	}

	for i, tt := range tests {
		mt.addResult(&http.Response{
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(tt.res)),
			Status:        "OK",
			StatusCode:    200,
			Body:          bodyReader(tt.res),
		}, nil)
		got, err := client.CreateHMACKey(ctx, projectID, serviceAccountEmail)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d: failed to match errors:\ngot:  %q\nwant: %q", i, err, tt.wantErr)
			}
			if got != nil {
				t.Errorf("#%d: unexpectedly got a non-nil result: %#v\n", i, got)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: got an unexpected error: %v", i, err)
			continue
		}

		if diff := testutil.Diff(got, tt.want); diff != "" {
			t.Errorf("#%d: got - want +\n\n%s", i, diff)
		}
	}

	// Lastly ensure that a blank service account will return an error.
	mt.addResult(&http.Response{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Status:     "OK",
		StatusCode: 200,
		Body:       bodyReader("{}"),
	}, nil)
	hk, err := client.CreateHMACKey(ctx, projectID, "")
	if err == nil {
		t.Fatal("Unexpectedly succeeded in creating a key using a blank service account email")
	}
	if !strings.Contains(err.Error(), "non-blank service account email") {
		t.Fatalf("Expected an error about a non-blank service account email: %v", err)
	}
	if hk != nil {
		t.Fatalf("Unexpectedly got back a created HMACKey: %#v", hk)
	}
}

func TestHMACKey_UpdateState(t *testing.T) {
	// This test ensures that updating the state can only
	// happen with either of Active or Inactive.

	mt := &mockTransport{}
	client := mockClient(t, mt)
	projectID := "hmackey-project-id"
	ctx := context.Background()

	hkh := client.HMACKeyHandle(projectID, "some-access-id")

	// 1. Ensure that invalid states are NOT accepted for an Update.
	invalidStates := []HMACState{"", Deleted, "active", "inactive", "foo_bar"}
	for _, invalidState := range invalidStates {
		t.Run("invalid-"+string(invalidState), func(t *testing.T) {
			_, err := hkh.Update(ctx, HMACKeyAttrsToUpdate{
				State: invalidState,
			})
			if err == nil {
				t.Fatal("Unexpectedly succeeded")
			}
			invalidStateMsg := fmt.Sprintf(`storage: invalid state %q for update, must be either "ACTIVE" or "INACTIVE"`, invalidState)
			if err.Error() != invalidStateMsg {
				t.Fatalf("Mismatched error: got:  %q\nwant: %q", err, invalidStateMsg)
			}
		})
	}

	// 2. Ensure that valid states for Update are accepted.
	validStates := []HMACState{Active, Inactive}
	for _, validState := range validStates {
		t.Run("valid-"+string(validState), func(t *testing.T) {
			resBody := fmt.Sprintf(`{
                                    "kind": "storage#hmacKeyMetadata",
                                    "state":%q,
                                    "timeCreated": "2019-07-11T12:11:33+00:00",
                                    "projectId": "project-id",
                                    "updated": "2019-07-11T12:13:33+00:00"
                                }
                            }`, validState)
			mt.addResult(&http.Response{
				ProtoMajor:    1,
				ProtoMinor:    1,
				ContentLength: int64(len(resBody)),
				Status:        "OK",
				StatusCode:    200,
				Body:          bodyReader(resBody),
			}, nil)

			hu, err := hkh.Update(ctx, HMACKeyAttrsToUpdate{
				State: validState,
			})
			if err != nil {
				t.Fatalf("Unexpected failure: %v", err)
			}
			if hu.State != validState {
				t.Fatalf("Unexpected updated state %q, expected %q", hu.State, validState)
			}
		})
	}
}
