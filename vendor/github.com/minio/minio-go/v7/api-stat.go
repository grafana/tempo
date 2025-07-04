/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import (
	"context"
	"net/http"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// BucketExists verifies if bucket exists and you have permission to access it. Allows for a Context to
// control cancellations and timeouts.
func (c *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return false, err
	}

	// Execute HEAD on bucketName.
	resp, err := c.executeMethod(ctx, http.MethodHead, requestMetadata{
		bucketName:       bucketName,
		contentSHA256Hex: emptySHA256Hex,
	})
	defer closeResponse(resp)
	if err != nil {
		if ToErrorResponse(err).Code == NoSuchBucket {
			return false, nil
		}
		return false, err
	}
	if resp != nil {
		resperr := httpRespToErrorResponse(resp, bucketName, "")
		if ToErrorResponse(resperr).Code == NoSuchBucket {
			return false, nil
		}
		if resp.StatusCode != http.StatusOK {
			return false, httpRespToErrorResponse(resp, bucketName, "")
		}
	}
	return true, nil
}

// StatObject verifies if object exists, you have permission to access it
// and returns information about the object.
func (c *Client) StatObject(ctx context.Context, bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return ObjectInfo{}, ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Code:       InvalidBucketName,
			Message:    err.Error(),
		}
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return ObjectInfo{}, ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Code:       XMinioInvalidObjectName,
			Message:    err.Error(),
		}
	}
	headers := opts.Header()
	if opts.Internal.ReplicationDeleteMarker {
		headers.Set(minIOBucketReplicationDeleteMarker, "true")
	}
	if opts.Internal.IsReplicationReadyForDeleteMarker {
		headers.Set(isMinioTgtReplicationReady, "true")
	}

	// Execute HEAD on objectName.
	resp, err := c.executeMethod(ctx, http.MethodHead, requestMetadata{
		bucketName:       bucketName,
		objectName:       objectName,
		queryValues:      opts.toQueryValues(),
		contentSHA256Hex: emptySHA256Hex,
		customHeader:     headers,
	})
	defer closeResponse(resp)
	if err != nil {
		return ObjectInfo{}, err
	}

	if resp != nil {
		deleteMarker := resp.Header.Get(amzDeleteMarker) == "true"
		replicationReady := resp.Header.Get(minioTgtReplicationReady) == "true"
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			if resp.StatusCode == http.StatusMethodNotAllowed && opts.VersionID != "" && deleteMarker {
				errResp := ErrorResponse{
					StatusCode: resp.StatusCode,
					Code:       MethodNotAllowed,
					Message:    s3ErrorResponseMap[MethodNotAllowed],
					BucketName: bucketName,
					Key:        objectName,
				}
				return ObjectInfo{
					VersionID:      resp.Header.Get(amzVersionID),
					IsDeleteMarker: deleteMarker,
				}, errResp
			}
			return ObjectInfo{
				VersionID:        resp.Header.Get(amzVersionID),
				IsDeleteMarker:   deleteMarker,
				ReplicationReady: replicationReady, // whether delete marker can be replicated
			}, httpRespToErrorResponse(resp, bucketName, objectName)
		}
	}

	return ToObjectInfo(bucketName, objectName, resp.Header)
}
