package models

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type TextObj struct {
	Text      string
	Sensative []SensativeWords
}

type SensativeWords struct {
	Text        string `json:"sensative_words" bson:"sensative_words" binding:"required"`         //這兩個已經做移除
	Apply       string `json:"apply_to" bson:"apply_to" binding:"required" validate:"ValidApply"` //這兩個已經做移除
	Policy      string `json:"policy" bson:"policy" validate:"ValidPolicy"`
	Gender      string `json:"gender" bson:"gender" validate:"ValidGender"`
	Replacement string `json:"replacement" bson:"replacement" validate:"ValidReplacement"`
}

type UserSensativeWordsCount struct {
	Sender   string   `json:"sender" bson:"sender"`
	Messages []string `json:"messages" bson:"messages" binding:"required"`
	Count    int      `json:"count" bson:"count" binding:"required"`
}

func ValidApply(fl validator.FieldLevel) bool {
	allowed := map[string]bool{
		"gender": true,
		"user":   true,
		"all":    true,
	}
	country := fl.Field().String()
	return allowed[country]
}

func ValidPolicy(fl validator.FieldLevel) bool {
	allowed := map[string]bool{
		"obfuscate": true,
		"replace":   true,
	}
	country := fl.Field().String()
	return allowed[country]
}

func ValidGender(fl validator.FieldLevel) bool {
	apply := fl.Parent().FieldByName("Apply").String()
	gender := fl.Parent().FieldByName("Gender").String()

	return true
	if apply == "gender" && (gender == "男" || gender == "女") {
		return true
	}
	if apply != "gender" {
		return true
	}
	return false
}

func ValidReplacement(fl validator.FieldLevel) bool {
	policy := fl.Parent().FieldByName("Policy").String()
	replacement := fl.Parent().FieldByName("replacement").String()
	if policy != "replace" {
		return true
	}
	return policy == "replace" && replacement != ""
}

func (t *TextObj) IsSensativeExist(obj SensativeWords) bool {
	for _, it := range t.Sensative {
		if it.Text == obj.Text && it.Apply == obj.Apply {
			return true
		}
	}
	return false
}

func (t *TextObj) IsPolicyValid(obj SensativeWords) bool {
	if obj.Policy == "" || (obj.Policy != "obfuscate" && obj.Policy != "replace") {
		return false
	}
	if obj.Policy == "replace" && obj.Replacement == "" {
		return false
	}
	return true
}

func (t *TextObj) IsGenderValid(obj SensativeWords) bool {
	fmt.Println(obj.Apply)
	fmt.Println(obj.Gender)
	if obj.Apply == "gender" && (obj.Gender == "男" || obj.Gender == "女") {
		return true
	}
	if obj.Apply != "gender" {
		return true
	}
	return false
}

func (t *TextObj) TestingText(text string) string {
	for _, it := range t.Sensative {
		//fmt.Println(it.Text)
		if it.Policy == "obfuscate" {
			text = strings.Replace(text, it.Text, "***", -1)
		} else if it.Policy == "replace" {
			text = strings.Replace(text, it.Text, it.Replacement, -1)
		}
	}
	return text
}
