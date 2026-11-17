package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ErrRemoteMedia identifies a remote media URL that could not be inlined.
var ErrRemoteMedia = errors.New("remote media fetch failed")

const (
	maxRemoteMediaBytes      = 10 << 20
	maxRemoteMediaTotalBytes = 20 << 20
	remoteMediaTimeout       = 10 * time.Second
	maxRemoteMediaRedirects  = 3
)

// MediaFetcher downloads a remote image and reports its media type.
type MediaFetcher interface {
	Fetch(ctx context.Context, url string) (mediaType string, data []byte, err error)
}

// RemoteMediaFetcher downloads images with a private-network guard (SSRF) and
// hard size/time limits.
type RemoteMediaFetcher struct {
	client *http.Client
}

// NewRemoteMediaFetcher creates the default guarded fetcher.
func NewRemoteMediaFetcher() *RemoteMediaFetcher {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, address := range addresses {
				if BlockedIP(address.IP) {
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr == nil {
				return nil, fmt.Errorf("address is not allowed")
			}
			return nil, lastErr
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: remoteMediaTimeout,
	}
	return &RemoteMediaFetcher{client: &http.Client{
		Transport: transport,
		Timeout:   remoteMediaTimeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRemoteMediaRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}}
}

// Fetch downloads an image, rejecting non-image payloads and oversized bodies.
func (f *RemoteMediaFetcher) Fetch(ctx context.Context, rawURL string) (string, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", nil, err
	}
	request.Header.Set("User-Agent", "dontstop-gateway/0.1 (+https://dontstop.alltomatos.dev.br)")
	request.Header.Set("Accept", "image/*")
	response, err := f.client.Do(request)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	mediaType := response.Header.Get("Content-Type")
	if index := strings.IndexByte(mediaType, ';'); index >= 0 {
		mediaType = mediaType[:index]
	}
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	if !strings.HasPrefix(mediaType, "image/") {
		return "", nil, fmt.Errorf("unsupported content type %q", mediaType)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxRemoteMediaBytes+1))
	if err != nil {
		return "", nil, err
	}
	if len(data) > maxRemoteMediaBytes {
		return "", nil, fmt.Errorf("image exceeds %d bytes", maxRemoteMediaBytes)
	}
	return mediaType, data, nil
}

// BlockedIP reports whether an address belongs to a network the gateway must
// not reach on behalf of a client.
func BlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127 {
			return true
		}
		if ipv4[0] == 0 {
			return true
		}
	}
	if len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc {
		return true
	}
	return false
}

// InlineRemoteMedia replaces http(s) image URLs with data URLs. It returns the
// adjusted request and how many images were inlined.
func InlineRemoteMedia(ctx context.Context, request ChatRequest, fetcher MediaFetcher) (ChatRequest, int, error) {
	if fetcher == nil {
		return request, 0, nil
	}
	total := 0
	inlined := 0
	messages := make([]ChatMessage, len(request.Messages))
	for index, message := range request.Messages {
		if message.Parts == nil {
			messages[index] = message
			continue
		}
		parts := make([]ContentPart, len(message.Parts))
		copy(parts, message.Parts)
		changed := false
		for partIndex := range parts {
			part := &parts[partIndex]
			if part.Type != ContentPartImageURL || part.ImageURL == nil {
				continue
			}
			rawURL := strings.TrimSpace(part.ImageURL.URL)
			if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
				continue
			}
			mediaType, data, err := fetcher.Fetch(ctx, rawURL)
			if err != nil {
				return ChatRequest{}, inlined, fmt.Errorf("%w: %v", ErrRemoteMedia, err)
			}
			total += len(data)
			if total > maxRemoteMediaTotalBytes {
				return ChatRequest{}, inlined, fmt.Errorf("%w: total media exceeds %d bytes", ErrRemoteMedia, maxRemoteMediaTotalBytes)
			}
			part.ImageURL = &ImageURL{
				URL:    "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data),
				Detail: part.ImageURL.Detail,
			}
			changed = true
			inlined++
		}
		if changed {
			message.Parts = parts
			message.Content = joinTextParts(parts)
		}
		messages[index] = message
	}
	if inlined == 0 {
		return request, 0, nil
	}
	request.Messages = messages
	return request, inlined, nil
}
