// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package receiver defines components that allows the collector to receive metrics, traces and logs.
//
// Receiver receives data from a source (either from a remote source via network
// or scrapes from a local host) and pushes the data to the pipelines it is attached
// to by calling the nextConsumer.Consume*() function.
//
// # Error Handling
//
// The nextConsumer.Consume*() function may return an error to indicate that the data
// was not accepted. There are 2 types of possible errors: Permanent and non-Permanent.
// The receiver must check the type of the error using IsPermanent() helper.
//
// If the error is Permanent, then the nextConsumer.Consume*() call should not be
// retried with the same data. This typically happens when the data cannot be
// serialized by the exporter that is attached to the pipeline or when the destination
// refuses the data because it cannot decode it. The receiver must indicate to
// the source from which it received the data that the received data was bad, if the
// receiving protocol allows to do that. In case of OTLP/HTTP for example, this means
// that HTTP 400 response is returned to the sender.
//
// If the error is non-Permanent then the nextConsumer.Consume*() call may be retried
// with the same data. This may be done by the receiver itself, however typically it is
// done by the original sender, after the receiver returns a response to the sender
// indicating that the Collector is currently overloaded and the request must be
// retried. In case of OTLP/HTTP for example, this means that HTTP 429 or 503 response
// is returned.
//
// # Acknowledgment Handling
//
// The receivers that receive data via a network protocol that support acknowledgments
// MUST follow this order of operations:
//   - Receive data from some sender (typically from a network).
//   - Push received data to the pipeline by calling nextConsumer.Consume*() function.
//   - Acknowledge successful data receipt to the sender if Consume*() succeeded or
//     return a failure to the sender if Consume*() returned an error.
//
// This ensures there are strong delivery guarantees once the data is acknowledged
// by the Collector.
package receiver // import "go.opentelemetry.io/collector/receiver"
