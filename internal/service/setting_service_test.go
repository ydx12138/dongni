package service

import (
	"blog/models"
	"blog/models/dto"
	"strings"
	"testing"
)

func TestSiteSettingFromRowsIncludesProfileAbout(t *testing.T) {
	settings := []models.Setting{
		{Key: settingProfileAbout, Value: "第一段\n\n第二段"},
	}

	result := siteSettingFromRows(settings)

	if result.ProfileAbout != "第一段\n\n第二段" {
		t.Fatalf("ProfileAbout = %q, want %q", result.ProfileAbout, "第一段\n\n第二段")
	}
}

func TestSiteSettingFromRowsLeavesProfileAboutEmptyWhenMissing(t *testing.T) {
	result := siteSettingFromRows(nil)

	if result.ProfileAbout != "" {
		t.Fatalf("ProfileAbout = %q, want empty", result.ProfileAbout)
	}
}

func TestValidateSiteSettingRejectsProfileAboutOver2000Characters(t *testing.T) {
	req := dto.UpdateSiteSettingRequest{
		SiteTitle:    "懂你",
		ProfileAbout: strings.Repeat("文", 2001),
	}

	if err := validateSiteSetting(req); err == nil {
		t.Fatal("validateSiteSetting() error = nil, want profile about length error")
	}
}
