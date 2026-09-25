package personname

import (
	"testing"
)

func TestNewPersonName(t *testing.T) {
	pn := NewPersonName("Somchai", "Jaidee")

	if pn.GivenName != "Somchai" {
		t.Errorf("expected GivenName 'Somchai', got %q", pn.GivenName)
	}
	if pn.Surname != "Jaidee" {
		t.Errorf("expected Surname 'Jaidee', got %q", pn.Surname)
	}
	if pn.Title != "" {
		t.Errorf("expected empty Title, got %q", pn.Title)
	}
	if pn.MiddleName != "" {
		t.Errorf("expected empty MiddleName, got %q", pn.MiddleName)
	}
}

func TestPersonName_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pn      PersonName
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid name",
			pn:      PersonName{GivenName: "Somchai", Surname: "Jaidee"},
			wantErr: false,
		},
		{
			name:    "valid with all fields",
			pn:      PersonName{Title: "Dr.", GivenName: "Anna", MiddleName: "Marie", Surname: "Mueller"},
			wantErr: false,
		},
		{
			name:    "empty given_name",
			pn:      PersonName{Title: "Mr.", GivenName: "", Surname: "Smith"},
			wantErr: true,
			errMsg:  "given_name must not be empty",
		},
		{
			name:    "empty surname",
			pn:      PersonName{GivenName: "John", Surname: ""},
			wantErr: true,
			errMsg:  "surname must not be empty",
		},
		{
			name:    "whitespace-only given_name",
			pn:      PersonName{GivenName: "   ", Surname: "Smith"},
			wantErr: true,
			errMsg:  "given_name must not be empty",
		},
		{
			name:    "whitespace-only surname",
			pn:      PersonName{GivenName: "John", Surname: "   "},
			wantErr: true,
			errMsg:  "surname must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pn.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestPersonName_IsZero(t *testing.T) {
	tests := []struct {
		name string
		pn   PersonName
		want bool
	}{
		{
			name: "zero value",
			pn:   PersonName{},
			want: true,
		},
		{
			name: "non-zero with given and surname",
			pn:   PersonName{GivenName: "Somchai", Surname: "Jaidee"},
			want: false,
		},
		{
			name: "non-zero with title only",
			pn:   PersonName{Title: "Dr."},
			want: false,
		},
		{
			name: "non-zero with middle name only",
			pn:   PersonName{MiddleName: "Marie"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pn.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPersonName_FullName(t *testing.T) {
	tests := []struct {
		name string
		pn   PersonName
		want string
	}{
		{
			name: "with middle name",
			pn:   PersonName{GivenName: "Anna", MiddleName: "Marie", Surname: "Mueller"},
			want: "Anna Marie Mueller",
		},
		{
			name: "without middle name",
			pn:   PersonName{GivenName: "Somchai", Surname: "Jaidee"},
			want: "Somchai Jaidee",
		},
		{
			name: "title is not included in full name",
			pn:   PersonName{Title: "Dr.", GivenName: "Anna", Surname: "Mueller"},
			want: "Anna Mueller",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pn.FullName()
			if got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPersonName_FormalName(t *testing.T) {
	tests := []struct {
		name string
		pn   PersonName
		want string
	}{
		{
			name: "with title",
			pn:   PersonName{Title: "Mr.", GivenName: "Somchai", Surname: "Jaidee"},
			want: "Mr. Jaidee",
		},
		{
			name: "without title",
			pn:   PersonName{GivenName: "Pavel", Surname: "Novak"},
			want: "Novak",
		},
		{
			name: "Mrs title",
			pn:   PersonName{Title: "Mrs.", GivenName: "Siriwan", Surname: "Thongkham"},
			want: "Mrs. Thongkham",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pn.FormalName()
			if got != tt.want {
				t.Errorf("FormalName() = %q, want %q", got, tt.want)
			}
		})
	}
}
