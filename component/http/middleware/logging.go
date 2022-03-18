package middleware

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type intervalType uint32

const (
	included intervalType = iota // '[' or ']'
	excluded                     // '(' or ')'
)

type StatusCodeLoggerHandler struct {
	codes []statusCode
}

func (s StatusCodeLoggerHandler) shouldLog(statusCode int) bool {
	for _, code := range s.codes {
		if code.isRange() {
			if code.rangeCodes.isIncluded(statusCode) {
				return true
			}
		} else {
			if statusCode == code.exactCode {
				return true
			}
		}
	}
	return false
}

func NewStatusCodeLoggerHandler(cfg string) (StatusCodeLoggerHandler, error) {
	cfg = strings.TrimSpace(cfg)
	if len(cfg) == 0 {
		return StatusCodeLoggerHandler{}, nil
	}

	splits := strings.Split(cfg, ";")
	codes := make([]statusCode, len(splits))

	for idx, split := range splits {
		i, err := strconv.Atoi(split)
		isNumber := err == nil

		if isNumber {
			codes[idx] = statusCode{
				exactCode: i,
			}
		} else {
			codeRange, err := parseRange(split)
			if err != nil {
				return StatusCodeLoggerHandler{}, fmt.Errorf("failed to parse status code range %q: %w", split, err)
			}

			codes[idx] = statusCode{
				rangeCodes: &codeRange,
			}
		}
	}
	return StatusCodeLoggerHandler{
		codes: codes,
	}, nil
}

func parseRange(s string) (statusCodeRange, error) {
	// Expected ASCII characters so no need to convert into runes
	if len(s) < 2 {
		return statusCodeRange{}, errors.New("range format error")
	}

	startInterval, err := parseStartInterval(s[0])
	if err != nil {
		return statusCodeRange{}, err
	}

	endInterval, err := parseEndInterval(s[len(s)-1])
	if err != nil {
		return statusCodeRange{}, err
	}

	codesWithoutIntervalTypes := s[1 : len(s)-1]

	splits := strings.Split(codesWithoutIntervalTypes, ",")
	if len(splits) != 2 {
		return statusCodeRange{}, fmt.Errorf("expected 2 status codes in the range, got %d", len(splits))
	}

	start, err := strconv.Atoi(splits[0])
	if err != nil {
		return statusCodeRange{}, fmt.Errorf("invalid range start %q", splits[0])
	}

	end, err := strconv.Atoi(splits[1])
	if err != nil {
		return statusCodeRange{}, fmt.Errorf("invalid range end %q", splits[1])
	}

	return statusCodeRange{
		start:         start,
		startInterval: startInterval,
		end:           end,
		endInterval:   endInterval,
	}, nil
}

func parseStartInterval(c uint8) (intervalType, error) {
	if c == '[' {
		return included, nil
	}
	if c == '(' {
		return excluded, nil
	}
	return 0, fmt.Errorf(`invalid interval type, expected [ or (, got %c`, c)
}

func parseEndInterval(c uint8) (intervalType, error) {
	if c == ']' {
		return included, nil
	}
	if c == ')' {
		return excluded, nil
	}
	return 0, fmt.Errorf(`invalid interval type, expected ] or ), got %c`, c)
}

type statusCode struct {
	exactCode  int
	rangeCodes *statusCodeRange
}

func (s statusCode) isRange() bool {
	return s.rangeCodes != nil
}

type statusCodeRange struct {
	start         int
	startInterval intervalType
	end           int
	endInterval   intervalType
}

func (s *statusCodeRange) isIncluded(statusCode int) bool {
	if s.startInterval == included && s.endInterval == included {
		return statusCode >= s.start && statusCode <= s.end
	}
	if s.startInterval == included && s.endInterval == excluded {
		return statusCode >= s.start && statusCode < s.end
	}
	if s.startInterval == excluded && s.endInterval == included {
		return statusCode > s.start && statusCode <= s.end
	}
	return statusCode > s.start && statusCode < s.end
}
