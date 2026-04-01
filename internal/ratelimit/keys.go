package ratelimit

import (
	"strings"

	"worksphere-api/pkg/validation"
)

func LoginIPKey(ip string) string {
	normalizedIP := strings.TrimSpace(ip)
	if normalizedIP == "" {
		normalizedIP = "unknown"
	}

	return "rl:login:ip:" + normalizedIP
}

func GlobalBurstKey(ip string) string {
	return "rl:global:burst:ip:" + normalizeIP(ip)
}

func GlobalSustainedKey(ip string) string {
	return "rl:global:sustained:ip:" + normalizeIP(ip)
}

func LoginEmailKey(email string) string {
	return "rl:login:email:" + validation.NormalizeEmail(email)
}

func ResendVerificationEmailKey(email string) string {
	return "rl:resend-verification:email:" + validation.NormalizeEmail(email)
}

func ResendVerificationIPKey(ip string) string {
	return "rl:resend-verification:ip:" + normalizeIP(ip)
}

func RegisterMinuteKey(ip string) string {
	normalizedIP := normalizeIP(ip)
	return "rl:register:ip:minute:" + normalizedIP
}

func RegisterHourKey(ip string) string {
	normalizedIP := normalizeIP(ip)
	return "rl:register:ip:hour:" + normalizedIP
}

func normalizeIP(ip string) string {
	normalizedIP := strings.TrimSpace(ip)
	if normalizedIP == "" {
		return "unknown"
	}

	return normalizedIP
}
