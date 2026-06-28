package controllers

import (
	"errors"
	"net/http"
	"supplierhub-backend/config"
	"supplierhub-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateReviewInput struct {
	OrderID string `json:"order_id" binding:"required"`
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
}

// CreateReview menangani pembuatan rating dan ulasan oleh UMKM
func CreateReview(c *gin.Context) {
	umkmID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input CreateReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input tidak valid. Rating harus bernilai antara 1 sampai 5."})
		return
	}

	// 1. Validasi keberadaan Pesanan (Order)
	var order models.Order
	if err := config.DB.Where("id = ?", input.OrderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memverifikasi pesanan"})
		}
		return
	}

	// 2. Pastikan pesanan adalah milik UMKM yang bersangkutan
	if order.UmkmID != umkmID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses untuk memberikan ulasan pada pesanan ini"})
		return
	}

	// 3. Pastikan pesanan berstatus completed
	if order.Status != models.OrderCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ulasan hanya dapat diberikan pada pesanan yang sudah selesai diterima"})
		return
	}

	// 4. Pastikan belum ada ulasan sebelumnya untuk order_id ini (satu ulasan per pesanan)
	var existingReview models.Review
	if err := config.DB.Where("order_id = ?", input.OrderID).First(&existingReview).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Anda sudah pernah memberikan ulasan untuk pesanan ini"})
		return
	}

	// 5. Buat entitas Review baru
	newReview := models.Review{
		OrderID:   order.ID,
		ProductID: order.ProductID,
		UmkmID:    umkmID,
		Rating:    input.Rating,
		Comment:   input.Comment,
	}

	if err := config.DB.Create(&newReview).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ulasan"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Ulasan berhasil dikirim!",
		"data":    newReview,
	})
}

// GetProductReviews mengambil daftar ulasan untuk suatu produk tertentu secara publik
func GetProductReviews(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID wajib disertakan"})
		return
	}

	var reviews []models.Review
	if err := config.DB.Preload("Umkm").Where("product_id = ?", productID).Order("created_at desc").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data ulasan"})
		return
	}

	// Format response yang bersih
	type ReviewResponse struct {
		ID        string `json:"id"`
		Rating    int    `json:"rating"`
		Comment   string `json:"comment"`
		CreatedAt string `json:"created_at"`
		UmkmName  string `json:"umkm_name"`
	}

	var responseList []ReviewResponse
	for _, r := range reviews {
		name := r.Umkm.BusinessName
		if name == "" {
			name = "UMKM Pembeli"
		}
		responseList = append(responseList, ReviewResponse{
			ID:        r.ID,
			Rating:    r.Rating,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
			UmkmName:  name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   responseList,
	})
}
