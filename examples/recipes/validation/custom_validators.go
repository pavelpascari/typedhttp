package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Custom validation patterns and examples
// Copy-paste ready for production use

// Initialize custom validators
func init() {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidation("username", validateUsername)
	v.RegisterValidation("phone", validatePhoneNumber)
	v.RegisterValidation("password_strength", validatePasswordStrength)
	v.RegisterValidation("slug", validateSlug)
	v.RegisterValidation("color_hex", validateColorHex)
	v.RegisterValidation("timezone", validateTimezone)
	v.RegisterValidation("credit_card", validateCreditCard)
	v.RegisterValidation("past_date", validatePastDate)
	v.RegisterValidation("future_date", validateFutureDate)
	v.RegisterValidation("business_hours", validateBusinessHours)
}

// Custom validation functions

func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	// Username: 3-20 chars, alphanumeric + underscore, starts with letter
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]{2,19}$`, username)
	return matched
}

func validatePhoneNumber(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// E.164 format: +1234567890 or simple validation
	matched, _ := regexp.MatchString(`^\+?[1-9]\d{1,14}$`, phone)
	return matched
}

func validatePasswordStrength(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// At least 8 characters
	if len(password) < 8 {
		return false
	}

	// Must contain at least one uppercase, lowercase, digit, and special char
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}

func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	// URL-friendly slug: lowercase, hyphens, no spaces
	matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, slug)
	return matched
}

func validateColorHex(fl validator.FieldLevel) bool {
	color := fl.Field().String()
	// Hex color: #123456 or #abc
	matched, _ := regexp.MatchString(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`, color)
	return matched
}

func validateTimezone(fl validator.FieldLevel) bool {
	tz := fl.Field().String()
	_, err := time.LoadLocation(tz)
	return err == nil
}

func validateCreditCard(fl validator.FieldLevel) bool {
	card := fl.Field().String()
	// Simple Luhn algorithm validation
	return luhnCheck(card)
}

func validatePastDate(fl validator.FieldLevel) bool {
	date := fl.Field().Interface().(time.Time)
	return date.Before(time.Now())
}

func validateFutureDate(fl validator.FieldLevel) bool {
	date := fl.Field().Interface().(time.Time)
	return date.After(time.Now())
}

func validateBusinessHours(fl validator.FieldLevel) bool {
	hours := fl.Field().String()
	// Format: "09:00-17:00"
	matched, _ := regexp.MatchString(`^([01]?[0-9]|2[0-3]):[0-5][0-9]-([01]?[0-9]|2[0-3]):[0-5][0-9]$`, hours)
	return matched
}

// Helper function for credit card validation (Luhn algorithm)
func luhnCheck(card string) bool {
	// Remove spaces and hyphens
	card = strings.ReplaceAll(card, " ", "")
	card = strings.ReplaceAll(card, "-", "")

	// Must be numeric and 13-19 digits
	if len(card) < 13 || len(card) > 19 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(card) - 1; i >= 0; i-- {
		n := int(card[i] - '0')
		if n < 0 || n > 9 {
			return false
		}

		if alternate {
			n *= 2
			if n > 9 {
				n = (n % 10) + 1
			}
		}

		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

// Example request types using custom validators

// User registration with comprehensive validation
type CreateUserRequest struct {
	Username        string `json:"username" validate:"required,username"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,password_strength"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
	Phone           string `json:"phone" validate:"required,phone"`
	FullName        string `json:"full_name" validate:"required,min=2,max=100"`
	DateOfBirth     string `json:"date_of_birth" validate:"required,datetime=2006-01-02"`
	Timezone        string `json:"timezone" validate:"required,timezone"`
}

// Business profile validation
type BusinessProfileRequest struct {
	CompanyName    string `json:"company_name" validate:"required,min=2,max=200"`
	Slug           string `json:"slug" validate:"required,slug,min=3,max=50"`
	Website        string `json:"website" validate:"omitempty,url"`
	Phone          string `json:"phone" validate:"required,phone"`
	BusinessHours  string `json:"business_hours" validate:"required,business_hours"`
	PrimaryColor   string `json:"primary_color" validate:"required,color_hex"`
	SecondaryColor string `json:"secondary_color" validate:"omitempty,color_hex"`
	Industry       string `json:"industry" validate:"required,oneof=tech healthcare finance retail education"`
}

// Payment method validation
type PaymentMethodRequest struct {
	Type          string `json:"type" validate:"required,oneof=credit_card debit_card paypal crypto"`
	CardNumber    string `json:"card_number" validate:"required_if=Type credit_card,required_if=Type debit_card,omitempty,credit_card"`
	ExpiryMonth   int    `json:"expiry_month" validate:"required_if=Type credit_card,required_if=Type debit_card,omitempty,min=1,max=12"`
	ExpiryYear    int    `json:"expiry_year" validate:"required_if=Type credit_card,required_if=Type debit_card,omitempty,min=2024"`
	CVV           string `json:"cvv" validate:"required_if=Type credit_card,required_if=Type debit_card,omitempty,len=3"`
	PayPalEmail   string `json:"paypal_email" validate:"required_if=Type paypal,omitempty,email"`
	CryptoAddress string `json:"crypto_address" validate:"required_if=Type crypto,omitempty,min=26,max=62"`
	CryptoType    string `json:"crypto_type" validate:"required_if=Type crypto,omitempty,oneof=bitcoin ethereum litecoin"`
}

// Event scheduling with date validation
type CreateEventRequest struct {
	Title        string    `json:"title" validate:"required,min=3,max=200"`
	Description  string    `json:"description" validate:"max=2000"`
	StartTime    time.Time `json:"start_time" validate:"required,future_date"`
	EndTime      time.Time `json:"end_time" validate:"required,gtfield=StartTime"`
	Location     string    `json:"location" validate:"required,min=5,max=300"`
	MaxAttendees int       `json:"max_attendees" validate:"required,min=1,max=10000"`
	IsPublic     bool      `json:"is_public"`
	Tags         []string  `json:"tags" validate:"max=10,dive,min=2,max=30"`
}

// Advanced validation with conditional logic
type UserPreferencesRequest struct {
	EmailNotifications  bool   `json:"email_notifications"`
	SMSNotifications    bool   `json:"sms_notifications"`
	Phone               string `json:"phone" validate:"required_if=SMSNotifications true,omitempty,phone"`
	MarketingEmails     bool   `json:"marketing_emails"`
	NewsletterFrequency string `json:"newsletter_frequency" validate:"required_if=MarketingEmails true,omitempty,oneof=daily weekly monthly"`
	Theme               string `json:"theme" validate:"required,oneof=light dark auto"`
	Language            string `json:"language" validate:"required,len=2"`
	Timezone            string `json:"timezone" validate:"required,timezone"`
	DateFormat          string `json:"date_format" validate:"required,oneof=MM/DD/YYYY DD/MM/YYYY YYYY-MM-DD"`
	TimeFormat          string `json:"time_format" validate:"required,oneof=12h 24h"`
}

// Handlers with validation error mapping

type CreateUserHandler struct {
	userService UserService
}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	// Additional business logic validation
	if err := h.validateBusinessRules(ctx, req); err != nil {
		return UserResponse{}, err
	}

	// Create user
	user, err := h.userService.CreateUser(ctx, req)
	if err != nil {
		return UserResponse{}, err
	}

	return UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
	}, nil
}

func (h *CreateUserHandler) validateBusinessRules(ctx context.Context, req CreateUserRequest) error {
	// Check if username is available
	exists, err := h.userService.UsernameExists(ctx, req.Username)
	if err != nil {
		return fmt.Errorf("failed to check username availability: %w", err)
	}
	if exists {
		return typedhttp.NewValidationError("Username already taken", map[string]string{
			"username": "already_exists",
		})
	}

	// Check if email is available
	exists, err = h.userService.EmailExists(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("failed to check email availability: %w", err)
	}
	if exists {
		return typedhttp.NewValidationError("Email already registered", map[string]string{
			"email": "already_exists",
		})
	}

	// Validate date of birth (must be at least 13 years old)
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return typedhttp.NewValidationError("Invalid date format", map[string]string{
			"date_of_birth": "invalid_format",
		})
	}

	thirteenYearsAgo := time.Now().AddDate(-13, 0, 0)
	if dob.After(thirteenYearsAgo) {
		return typedhttp.NewValidationError("Must be at least 13 years old", map[string]string{
			"date_of_birth": "too_young",
		})
	}

	return nil
}

// Cross-field validation example
type BookingRequest struct {
	CheckInDate  time.Time `json:"check_in_date" validate:"required,future_date"`
	CheckOutDate time.Time `json:"check_out_date" validate:"required,gtfield=CheckInDate"`
	Guests       int       `json:"guests" validate:"required,min=1,max=20"`
	RoomType     string    `json:"room_type" validate:"required,oneof=standard deluxe suite"`
	SpecialReqs  string    `json:"special_requirements" validate:"max=500"`
}

type BookingHandler struct{}

func (h *BookingHandler) Handle(ctx context.Context, req BookingRequest) (BookingResponse, error) {
	// Additional validation: minimum stay duration
	duration := req.CheckOutDate.Sub(req.CheckInDate)
	if duration < 24*time.Hour {
		return BookingResponse{}, typedhttp.NewValidationError("Minimum stay is 1 night", map[string]string{
			"check_out_date": "minimum_stay_required",
		})
	}

	// Maximum advance booking
	maxAdvanceBooking := time.Now().AddDate(1, 0, 0) // 1 year
	if req.CheckInDate.After(maxAdvanceBooking) {
		return BookingResponse{}, typedhttp.NewValidationError("Cannot book more than 1 year in advance", map[string]string{
			"check_in_date": "too_far_in_future",
		})
	}

	// Process booking...
	return BookingResponse{
		ID:           "booking_123",
		CheckInDate:  req.CheckInDate,
		CheckOutDate: req.CheckOutDate,
		Status:       "confirmed",
	}, nil
}

// Response types
type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type BookingResponse struct {
	ID           string    `json:"id"`
	CheckInDate  time.Time `json:"check_in_date"`
	CheckOutDate time.Time `json:"check_out_date"`
	Status       string    `json:"status"`
}

// Mock service interface
type UserService interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	EmailExists(ctx context.Context, email string) (bool, error)
}

type User struct {
	ID       string
	Username string
	Email    string
	FullName string
}

// Example validation middleware
type ValidationErrorResponse struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

// Enhanced error handler that provides detailed validation feedback
func EnhancedValidationErrorHandler(err error) (int, interface{}) {
	if validationErr, ok := err.(*typedhttp.ValidationError); ok {
		return 400, ValidationErrorResponse{
			Message: validationErr.Message,
			Errors:  validationErr.Fields,
		}
	}
	return 500, map[string]string{"error": "Internal server error"}
}

// Router setup with validation
func SetupValidationRoutes(router *typedhttp.TypedRouter, userService UserService) {
	createUserHandler := &CreateUserHandler{userService: userService}
	bookingHandler := &BookingHandler{}

	// Register handlers with enhanced validation
	typedhttp.POST(router, "/users", createUserHandler)
	typedhttp.POST(router, "/bookings", bookingHandler)
}
