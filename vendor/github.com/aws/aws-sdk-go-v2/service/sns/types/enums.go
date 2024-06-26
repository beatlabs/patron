// Code generated by smithy-go-codegen DO NOT EDIT.

package types

type LanguageCodeString string

// Enum values for LanguageCodeString
const (
	LanguageCodeStringEnUs  LanguageCodeString = "en-US"
	LanguageCodeStringEnGb  LanguageCodeString = "en-GB"
	LanguageCodeStringEs419 LanguageCodeString = "es-419"
	LanguageCodeStringEsEs  LanguageCodeString = "es-ES"
	LanguageCodeStringDeDe  LanguageCodeString = "de-DE"
	LanguageCodeStringFrCa  LanguageCodeString = "fr-CA"
	LanguageCodeStringFrFr  LanguageCodeString = "fr-FR"
	LanguageCodeStringItIt  LanguageCodeString = "it-IT"
	LanguageCodeStringJpJp  LanguageCodeString = "ja-JP"
	LanguageCodeStringPtBr  LanguageCodeString = "pt-BR"
	LanguageCodeStringKrKr  LanguageCodeString = "kr-KR"
	LanguageCodeStringZhCn  LanguageCodeString = "zh-CN"
	LanguageCodeStringZhTw  LanguageCodeString = "zh-TW"
)

// Values returns all known values for LanguageCodeString. Note that this can be
// expanded in the future, and so it is only as up to date as the client.
//
// The ordering of this slice is not guaranteed to be stable across updates.
func (LanguageCodeString) Values() []LanguageCodeString {
	return []LanguageCodeString{
		"en-US",
		"en-GB",
		"es-419",
		"es-ES",
		"de-DE",
		"fr-CA",
		"fr-FR",
		"it-IT",
		"ja-JP",
		"pt-BR",
		"kr-KR",
		"zh-CN",
		"zh-TW",
	}
}

type NumberCapability string

// Enum values for NumberCapability
const (
	NumberCapabilitySms   NumberCapability = "SMS"
	NumberCapabilityMms   NumberCapability = "MMS"
	NumberCapabilityVoice NumberCapability = "VOICE"
)

// Values returns all known values for NumberCapability. Note that this can be
// expanded in the future, and so it is only as up to date as the client.
//
// The ordering of this slice is not guaranteed to be stable across updates.
func (NumberCapability) Values() []NumberCapability {
	return []NumberCapability{
		"SMS",
		"MMS",
		"VOICE",
	}
}

type RouteType string

// Enum values for RouteType
const (
	RouteTypeTransactional RouteType = "Transactional"
	RouteTypePromotional   RouteType = "Promotional"
	RouteTypePremium       RouteType = "Premium"
)

// Values returns all known values for RouteType. Note that this can be expanded
// in the future, and so it is only as up to date as the client.
//
// The ordering of this slice is not guaranteed to be stable across updates.
func (RouteType) Values() []RouteType {
	return []RouteType{
		"Transactional",
		"Promotional",
		"Premium",
	}
}

type SMSSandboxPhoneNumberVerificationStatus string

// Enum values for SMSSandboxPhoneNumberVerificationStatus
const (
	SMSSandboxPhoneNumberVerificationStatusPending  SMSSandboxPhoneNumberVerificationStatus = "Pending"
	SMSSandboxPhoneNumberVerificationStatusVerified SMSSandboxPhoneNumberVerificationStatus = "Verified"
)

// Values returns all known values for SMSSandboxPhoneNumberVerificationStatus.
// Note that this can be expanded in the future, and so it is only as up to date as
// the client.
//
// The ordering of this slice is not guaranteed to be stable across updates.
func (SMSSandboxPhoneNumberVerificationStatus) Values() []SMSSandboxPhoneNumberVerificationStatus {
	return []SMSSandboxPhoneNumberVerificationStatus{
		"Pending",
		"Verified",
	}
}
