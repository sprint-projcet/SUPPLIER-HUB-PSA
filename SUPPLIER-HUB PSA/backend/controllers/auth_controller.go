package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"supplierhub-backend/config"
	"supplierhub-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const googleOAuthStateCookie = "supplierhub_google_oauth_state"

type googleUserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   `json:"verified_email"`
}

func generateOAuthState() (string, error) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(stateBytes), nil
}

func createAuthToken(user models.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super_secret_key_supplierhub"
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"sub":     user.ID,
		"role":    string(user.Role),
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func sendOAuthPopupMessage(c *gin.Context, payload gin.H) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat response OAuth"})
		return
	}

	htmlContent := fmt.Sprintf(`<!doctype html>
<html lang="id">
<body>
<script>
	const payload = %s;
	if (window.opener && !window.opener.closed) {
		window.opener.postMessage(payload, "*");
	}
	window.close();
</script>
</body>
</html>`, payloadJSON)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}

// DTO untuk input form Register
type RegisterInput struct {
	BusinessName string `json:"business_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=6"`
	Role         string `json:"role" binding:"required"`
}

// DTO untuk input form Login
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register menangani pendaftaran awal untuk UMKM dan Supplier
func Register(c *gin.Context) {
	// Parse fields dari form-data
	businessName := c.PostForm("business_name")
	email := c.PostForm("email")
	password := c.PostForm("password")
	role := c.PostForm("role")
	address := c.PostForm("address")
	category := c.PostForm("category")
	region := c.PostForm("region")

	// 1. Validasi Input
	isGoogle := c.PostForm("is_google") == "true"
	if businessName == "" || email == "" || role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kolom Nama, Email, dan Role wajib diisi!"})
		return
	}
	if !isGoogle && password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password wajib diisi!"})
		return
	}

	// 2. Cek apakah email sudah terdaftar
	var existingUser models.User
	if err := config.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar!"})
		return
	}

	// 3. File Upload Handling (Dokumen)
	var documentURL string
	file, err := c.FormFile("document")
	if err == nil {
		// Buat folder jika belum ada
		os.MkdirAll("uploads/documents", os.ModePerm)

		// Gunakan timestamp untuk nama file yang unik
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
		filepath := "uploads/documents/" + filename

		if err := c.SaveUploadedFile(file, filepath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan dokumen legalitas"})
			return
		}
		documentURL = filepath
	} else if role == "supplier" {
		// Supplier wajib upload dokumen
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dokumen legalitas (SIUP/Akta) wajib diunggah untuk pendaftaran supplier"})
		return
	}

	// 4. Hash Password (atau dummy jika Google)
	var hashedPassword []byte
	var errHash error
	if isGoogle {
		hashedPassword, errHash = bcrypt.GenerateFromPassword([]byte("google-oauth-dummy-pass"), bcrypt.DefaultCost)
	} else {
		hashedPassword, errHash = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	}

	if errHash != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}

	// 5. Simpan ke Database
	newUser := models.User{
		BusinessName: businessName,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         models.Role(role),
		Address:      address,
		Category:     category,
		Region:       region,
		DocumentURL:  documentURL,
		Status:       "pending", // Default status, mungkin butuh verifikasi admin nanti
	}

	if err := config.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data pengguna"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registrasi berhasil!",
		"user": gin.H{
			"id":            newUser.ID,
			"business_name": newUser.BusinessName,
			"email":         newUser.Email,
			"role":          newUser.Role,
		},
	})
}

// Login memvalidasi kredensial dan menerbitkan JWT Token
func Login(c *gin.Context) {
	var input LoginInput

	// 1. Validasi Input JSON
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid atau kurang lengkap"})
		return
	}

	// 2. Cari user berdasarkan Email
	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	// 3. Verifikasi Password dengan Hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	// 4. Cek Status User (Optional: Jangan ijinkan login jika status suspended)
	if user.Status == "suspended" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun Anda ditangguhkan"})
		return
	}

	// 5. Generate JWT Token
	tokenString, err := createAuthToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token autentikasi"})
		return
	}

	// 6. Kirim Respons Berhasil
	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"token":   tokenString,
		"role":    user.Role,
		"user": gin.H{
			"id":            user.ID,
			"business_name": user.BusinessName,
			"email":         user.Email,
			"address":       user.Address,
			"category":      user.Category,
			"region":        user.Region,
			"pic_name":      user.PICName,
			"phone":         user.Phone,
			"status":        user.Status,
		},
	})
}

// GoogleLogin mengarahkan user ke halaman login Google
func GoogleLogin(c *gin.Context) {
	if !config.IsGoogleOAuthConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google OAuth belum dikonfigurasi. Pastikan GOOGLE_CLIENT_ID dan GOOGLE_CLIENT_SECRET tersedia."})
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat state OAuth"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(googleOAuthStateCookie, state, 300, "/", "", false, true)

	url := config.GoogleOAuthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback menangani response dari Google setelah login
func GoogleCallback(c *gin.Context) {
	if !config.IsGoogleOAuthConfigured() {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Google OAuth belum dikonfigurasi.",
		})
		return
	}

	state := c.Query("state")
	savedState, err := c.Cookie(googleOAuthStateCookie)
	c.SetCookie(googleOAuthStateCookie, "", -1, "/", "", false, true)

	if err != nil || state == "" || savedState == "" || state != savedState {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Sesi login Google tidak valid atau sudah kedaluwarsa. Silakan coba lagi.",
		})
		return
	}

	if oauthError := c.Query("error"); oauthError != "" {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Login Google dibatalkan atau gagal: " + oauthError,
		})
		return
	}

	code := c.Query("code")
	if code == "" {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Kode otorisasi Google tidak ditemukan.",
		})
		return
	}

	token, err := config.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Gagal menukar token Google: " + err.Error(),
		})
		return
	}

	client := config.GoogleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Gagal mendapatkan info user dari Google.",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Google mengembalikan response userinfo yang tidak valid.",
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Gagal membaca response info user.",
		})
		return
	}

	var googleUser googleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Gagal parsing info user.",
		})
		return
	}

	if googleUser.Email == "" {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Email tidak ditemukan dari akun Google.",
		})
		return
	}

	if !googleUser.VerifiedEmail {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Email Google belum terverifikasi.",
		})
		return
	}

	// Cek user di DB
	var user models.User
	if err := config.DB.Where("email = ?", googleUser.Email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			sendOAuthPopupMessage(c, gin.H{
				"type":  "GOOGLE_LOGIN_ERROR",
				"error": "Gagal membaca data pengguna.",
			})
			return
		}

		// User belum ada, instruksikan frontend untuk menampilkan popup lengkapi profil & pilih role
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_NEEDS_REGISTRATION",
			"email": googleUser.Email,
			"name":  googleUser.Name,
		})
		return
	}

	// Generate JWT Token (sama seperti fungsi Login)
	tokenString, err := createAuthToken(user)
	if err != nil {
		sendOAuthPopupMessage(c, gin.H{
			"type":  "GOOGLE_LOGIN_ERROR",
			"error": "Gagal membuat token autentikasi.",
		})
		return
	}

	sendOAuthPopupMessage(c, gin.H{
		"type":  "GOOGLE_LOGIN_SUCCESS",
		"token": tokenString,
		"role":  string(user.Role),
		"user": gin.H{
			"id":            user.ID,
			"business_name": user.BusinessName,
			"email":         user.Email,
			"address":       user.Address,
			"category":      user.Category,
			"region":        user.Region,
			"pic_name":      user.PICName,
			"phone":         user.Phone,
			"status":        user.Status,
		},
	})
}
