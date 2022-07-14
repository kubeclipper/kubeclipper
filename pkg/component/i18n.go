/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package component

import (
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Lang string

const (
	English Lang = "en"
	Chinese Lang = "zh"
)

type I18nMessage struct {
	ID      string
	English string
	Chinese string
}

type I18nMessages []I18nMessage

var bundle = i18n.NewBundle(language.English)

func AddI18nMessages(msgs I18nMessages) error {
	for _, msg := range msgs {
		if msg.English == "" {
			return fmt.Errorf("english message can not be empty")
		}
		if msg.Chinese == "" {
			msg.Chinese = msg.English
		}
		if err := bundle.AddMessages(language.English, i18n.MustNewMessage(map[string]string{
			"ID":    msg.ID,
			"Other": msg.English,
		})); err != nil {
			return err
		}
		if err := bundle.AddMessages(language.Chinese, i18n.MustNewMessage(map[string]string{
			"ID":    msg.ID,
			"Other": msg.Chinese,
		})); err != nil {
			return err
		}
	}
	return nil
}

func GetLocalizer(lang Lang) *i18n.Localizer {
	bundle.LanguageTags()
	return i18n.NewLocalizer(bundle, string(lang))
}
