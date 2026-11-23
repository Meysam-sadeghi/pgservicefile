// Package pgservicefile is a parser for PostgreSQL service files (e.g. .pg_service.conf).
package pgservicefile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	maxServicefileBytes     = 4 << 20   // 4 MiB upper bound on servicefile size
	maxServicefileLineBytes = 64 * 1024 // maximum line length
	maxServices             = 10_000    // maximum number of service sections
	maxSettingsPerService   = 4096      // maximum keys per service
)

type Service struct {
	Name     string
	Settings map[string]string
}

type Servicefile struct {
	Services       []*Service
	servicesByName map[string]*Service
}

// GetService returns the named service.
func (sf *Servicefile) GetService(name string) (*Service, error) {
	if sf == nil || sf.servicesByName == nil {
		return nil, errors.New("not found")
	}
	service, present := sf.servicesByName[name]
	if !present {
		return nil, errors.New("not found")
	}
	return service, nil
}

// ReadServicefile reads the file at path and parses it into a Servicefile.
func ReadServicefile(path string) (*Servicefile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("pgservicefile: file must be a regular file")
	}
	if info.Size() > maxServicefileBytes {
		return nil, fmt.Errorf("pgservicefile: file size %d exceeds maximum %d bytes", info.Size(), maxServicefileBytes)
	}

	return ParseServicefile(f)
}

// ParseServicefile reads r and parses it into a Servicefile.
func ParseServicefile(r io.Reader) (*Servicefile, error) {
	if r == nil {
		return nil, errors.New("pgservicefile: reader must not be nil")
	}

	servicefile := &Servicefile{}
	lr := &io.LimitedReader{R: r, N: maxServicefileBytes + 1}

	var service *Service
	scanner := bufio.NewScanner(lr)
	scanner.Buffer(make([]byte, 0, 4096), maxServicefileLineBytes)
	lineNum := 0
	for scanner.Scan() {
		lineNum += 1
		line := scanner.Text()
		if strings.IndexByte(line, 0) >= 0 {
			return nil, fmt.Errorf("forbidden NUL byte on line %d", lineNum)
		}
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			// ignore comments and empty lines
		} else if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				return nil, fmt.Errorf("empty section name on line %d", lineNum)
			}
			if len(servicefile.Services) >= maxServices {
				return nil, fmt.Errorf("service count exceeds maximum %d", maxServices)
			}
			service = &Service{Name: name, Settings: make(map[string]string)}
			servicefile.Services = append(servicefile.Services, service)
		} else if service != nil {
			rawKey, rawVal, ok := strings.Cut(line, "=")
			if !ok {
				return nil, fmt.Errorf("unable to parse line %d", lineNum)
			}

			key := strings.TrimSpace(rawKey)
			if key == "" {
				return nil, fmt.Errorf("empty key on line %d", lineNum)
			}
			value := strings.TrimSpace(rawVal)

			if _, exists := service.Settings[key]; !exists && len(service.Settings) >= maxSettingsPerService {
				return nil, fmt.Errorf("settings count in service %q exceeds maximum %d", service.Name, maxSettingsPerService)
			}
			service.Settings[key] = value
		} else {
			return nil, fmt.Errorf("line %d is not in a section", lineNum)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if lr.N == 0 {
		return nil, fmt.Errorf("pgservicefile: input exceeds maximum %d bytes", maxServicefileBytes)
	}

	servicefile.servicesByName = make(map[string]*Service, len(servicefile.Services))
	for _, service := range servicefile.Services {
		servicefile.servicesByName[service.Name] = service
	}

	return servicefile, nil
}
