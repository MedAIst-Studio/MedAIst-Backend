package domain

import "errors"

type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(message string, code int) *Error {
	return &Error{
		Message: message,
		Code:    code,
	}
}

const (
	CodeBadRequest          = 400
	CodeUnauthorized        = 401
	CodeForbidden           = 403
	CodeNotFound            = 404
	CodeConflict            = 409
	CodeUnprocessableEntity = 422
	CodeInternalServerError = 500
	CodeServiceUnavailable  = 503
)

var (
	ErrMissingAuthHeader     = errors.New("Yetkilendirme başlığı eksik")
	ErrInvalidAuthHeader     = errors.New("Yetkilendirme başlığı formatı geçersiz")
	ErrInvalidToken          = errors.New("Geçersiz token yapısı")
	ErrForbidden             = errors.New("Erişim yasak")
	ErrExpiredToken          = errors.New("Token süresi dolmuş")
	ErrInvalidTokenSignature = errors.New("Token imzası geçersiz")
	ErrNotExistRefreshToken  = errors.New("Yenileme token'ı mevcut değil")

	ErrNotPDFFile       = errors.New("Dosya PDF formatında olmalıdır")
	ErrFileSizeExceeded = errors.New("Dosya boyutu 10 MB'ı geçemez")

	ErrEmptyUsername      = errors.New("Ad soyad boş olamaz")
	ErrPasswordTooShort   = errors.New("Parola en az 8 karakter olmalıdır")
	ErrInvalidRequest     = errors.New("Geçersiz istek")
	ErrUnauthorized       = errors.New("Yetkisiz erişim")
	ErrInvalidCredentials = errors.New("Geçersiz kimlik bilgileri")
	ErrTooShortUsername   = errors.New("Ad soyad en az 3 karakter olmalıdır")
	ErrTodoNotFound       = errors.New("Yapılacak öğe bulunamadı")

	ErrEmptyTitle          = errors.New("Başlık boş olamaz")
	ErrUserIdCannotBeEmpty = errors.New("Kullanıcı ID'si boş olamaz")
	ErrTitleTooLong        = errors.New("Başlık 100 karakteri geçemez")
	ErrTitleTooShort       = errors.New("Başlık en az 3 karakter olmalıdır")

	ErrUserAlreadyExists = errors.New("Kullanıcı zaten mevcut")
	ErrNoRows            = errors.New("Sonuçta satır bulunamadı")
	ErrEmailNotFound     = errors.New("E-posta bulunamadı")
	ErrUserNotFound      = errors.New("Kullanıcı bulunamadı")
	ErrInvalidUserID     = errors.New("Geçersiz kullanıcı ID'si")

	ErrEmailAlreadyVerified = errors.New("E-posta zaten doğrulanmış")
	ErrEmailAlreadyExists   = errors.New("E-posta zaten kayıtlı")
	ErrMissingRefreshToken  = errors.New("Yenileme token'ı eksik")

	ErrInfoNotFoundInPDF       = errors.New("PDF'de aradığınız bilgi bulunamadı, lütfen farklı bir soru deneyin")
	ErrNoContentToGenerateQuiz = errors.New("Quiz oluşturmak için yeterli içerik bulunamadı")
	ErrChatNotFound            = errors.New("Sohbet mevcut değil")
	ErrFileNotFound            = errors.New("Dosya mevcut değil")
	ErrInternalServer          = errors.New("Sunucu hatası")
)
