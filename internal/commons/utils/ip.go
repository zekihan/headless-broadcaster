package utils

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

var ErrIPNotFound = errors.New("IP not found")

func GetIP(r *http.Request) (string, error) {
	errs := make([]error, 0)

	ip, err := getIPFromXForwardedFor(r)
	if err == nil {
		return ip, nil
	}

	errs = append(errs, err)

	ip, err = getIPFromXRealIP(r)
	if err == nil {
		return ip, nil
	}

	errs = append(errs, err)

	ip, err = getIPFromRemoteAddr(r)
	if err == nil {
		return ip, nil
	}

	errs = append(errs, err)

	return "", errors.Join(errs...)
}

func getIPFromXForwardedFor(r *http.Request) (string, error) {
	ips := r.Header.Get("X-Forwarded-For")
	splitIps := strings.Split(ips, ",")

	if len(splitIps) > 0 {
		netIP := net.ParseIP(strings.TrimSpace(splitIps[0]))
		if netIP != nil {
			return netIP.String(), nil
		}
	}

	return "", ErrIPNotFound
}

func getIPFromXRealIP(r *http.Request) (string, error) {
	ip := r.Header.Get("X-Real-IP")

	netIP := net.ParseIP(ip)
	if netIP != nil {
		return netIP.String(), nil
	}

	return "", ErrIPNotFound
}

func getIPFromRemoteAddr(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("failed to split remote address: %w", err)
	}

	netIP := net.ParseIP(ip)
	if netIP != nil {
		ip := netIP.String()
		if ip == "::1" {
			return "127.0.0.1", nil
		}

		return ip, nil
	}

	return "", ErrIPNotFound
}
