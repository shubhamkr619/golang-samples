// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command livecaption_from_file streams a local audio file to
// Google Speech API and outputs the transcript.

package main

// [START speech_transcribe_streaming]
import (
	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var project = ""

const location = "global"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s <AUDIOFILE>\n", os.Args[0], filepath.Base(os.Args[1]))
		fmt.Fprintf(os.Stderr, "<AUDIOFILE> must be a path to a local audio file. Audio file must be a 16-bit signed little-endian encoded with a sample rate of 16000.\n")

	}
	flag.Parse()
	if len(flag.Args()) != 2 {
		log.Fatal("Please pass path to your project_id and local audio file as a command line argument")
	}
	audioFile := flag.Arg(1)
	project = flag.Arg(0)

	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Send the initial configuration message.
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		Recognizer: fmt.Sprintf("projects/%s/locations/%s/recognizers/_", project, location),
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					// In case of specific file encoding , so specify the decoding config.
					//DecodingConfig: &speechpb.RecognitionConfig_AutoDecodingConfig{},
					DecodingConfig: &speechpb.RecognitionConfig_ExplicitDecodingConfig{
						ExplicitDecodingConfig: &speechpb.ExplicitDecodingConfig{
							Encoding:          speechpb.ExplicitDecodingConfig_LINEAR16,
							SampleRateHertz:   16000,
							AudioChannelCount: 1,
						},
					},
					Model:         "long",
					LanguageCodes: []string{"en-US"},
					Features: &speechpb.RecognitionFeatures{
						MaxAlternatives: 2,
					},
				},
				StreamingFeatures: &speechpb.StreamingRecognitionFeatures{InterimResults: true},
			},
		},
	}); err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(audioFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				if err := stream.Send(&speechpb.StreamingRecognizeRequest{
					Recognizer: fmt.Sprintf("projects/%s/locations/%s/recognizers/_", project, location),
					StreamingRequest: &speechpb.StreamingRecognizeRequest_Audio{
						Audio: buf[:n],
					},
				}); err != nil {
					log.Printf("Could not send audio: %v", err)
				}
			}
			if err == io.EOF {
				// Nothing else to pipe, close the stream.
				if err := stream.CloseSend(); err != nil {
					log.Fatalf("Could not close stream: %v", err)
				}
				return
			}
			if err != nil {
				log.Printf("Could not read from %s: %v", audioFile, err)
				continue
			}
		}
	}()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			print("Recv break")
			break
		}
		if err != nil {
			log.Fatalf("Cannot stream results: %v", err)
		}
		for _, result := range resp.Results {
			if len(result.Alternatives) > 0 {
				if result.IsFinal == true {
					log.Println("result alternatives", result.Alternatives[0].Transcript, result.IsFinal)
				}

			}
		}
	}
}

// [END speech_transcribe_streaming]
