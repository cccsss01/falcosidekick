// SPDX-License-Identifier: Apache-2.0
/*
Copyright (C) 2023 The Falco Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package outputs

import (
	"bytes"
	"log"
	"strings"

	"github.com/falcosecurity/falcosidekick/types"
)

type header struct {
	Title    string `json:"title"`
	SubTitle string `json:"subtitle"`
}

type keyValue struct {
	TopLabel string `json:"topLabel"`
	Content  string `json:"content"`
}

type widget struct {
	KeyValue keyValue `json:"keyValue,omitempty"`
}

type section struct {
	Widgets []widget `json:"widgets"`
}

type card struct {
	Header   header    `json:"header,omitempty"`
	Sections []section `json:"sections,omitempty"`
}

type googlechatPayload struct {
	Text  string `json:"text,omitempty"`
	Cards []card `json:"cards,omitempty"`
}

func newGooglechatPayload(falcopayload types.FalcoPayload, config *types.Configuration) googlechatPayload {
	var messageText string
	widgets := []widget{}

	if config.Googlechat.MessageFormatTemplate != nil {
		buf := &bytes.Buffer{}
		if err := config.Googlechat.MessageFormatTemplate.Execute(buf, falcopayload); err != nil {
			log.Printf("[ERROR] : GoogleChat - Error expanding Google Chat message %v", err)
		} else {
			messageText = buf.String()
		}
	}

	if config.Googlechat.OutputFormat == Text {
		return googlechatPayload{
			Text: messageText,
		}
	}

	for _, i := range getSortedStringKeys(falcopayload.OutputFields) {
		widgets = append(widgets, widget{
			KeyValue: keyValue{
				TopLabel: i,
				Content:  falcopayload.OutputFields[i].(string),
			},
		})
	}

	widgets = append(widgets, widget{KeyValue: keyValue{"rule", falcopayload.Rule}})
	widgets = append(widgets, widget{KeyValue: keyValue{"priority", falcopayload.Priority.String()}})
	widgets = append(widgets, widget{KeyValue: keyValue{"source", falcopayload.Source}})

	if falcopayload.Hostname != "" {
		widgets = append(widgets, widget{KeyValue: keyValue{Hostname, falcopayload.Hostname}})
	}

	if len(falcopayload.Tags) != 0 {
		widgets = append(widgets, widget{
			KeyValue: keyValue{
				TopLabel: "tags",
				Content:  strings.Join(falcopayload.Tags, ", "),
			},
		})
	}

	widgets = append(widgets, widget{KeyValue: keyValue{"time", falcopayload.Time.String()}})

	return googlechatPayload{
		Text: messageText,
		Cards: []card{
			{
				Sections: []section{
					{Widgets: widgets},
				},
			},
		},
	}
}

// GooglechatPost posts event to Google Chat
func (c *Client) GooglechatPost(falcopayload types.FalcoPayload) {
	c.Stats.GoogleChat.Add(Total, 1)

	err := c.Post(newGooglechatPayload(falcopayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:googlechat", "status:error"})
		c.Stats.GoogleChat.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "googlechat", "status": Error}).Inc()
		log.Printf("[ERROR] : GoogleChat - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:googlechat", "status:ok"})
	c.Stats.GoogleChat.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "googlechat", "status": OK}).Inc()
}
