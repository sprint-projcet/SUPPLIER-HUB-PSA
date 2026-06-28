package controllers

import (
	"errors"
	"net/http"
	"strings"

	"supplierhub-backend/config"
	"supplierhub-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createWishlistInput struct {
	BahanBakuID string `json:"bahan_baku_id"`
	ProductID   string `json:"product_id"`
}

func AddWishlistItem(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var input createWishlistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data wishlist tidak valid"})
		return
	}

	productID := strings.TrimSpace(input.BahanBakuID)
	if productID == "" {
		productID = strings.TrimSpace(input.ProductID)
	}
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bahan_baku_id wajib diisi"})
		return
	}

	var product models.Product
	if err := config.DB.First(&product, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk bahan baku tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi produk"})
		return
	}

	var existing models.Wishlist
	if err := config.DB.Preload("Product").Where("user_id = ? AND bahan_baku_id = ?", userID, productID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Produk sudah ada di wishlist",
			"data":    existing,
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengecek wishlist"})
		return
	}

	wishlist := models.Wishlist{
		UserID:      userID,
		BahanBakuID: product.ID,
	}
	if err := config.DB.Create(&wishlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan wishlist"})
		return
	}

	if err := config.DB.Preload("Product.Supplier").First(&wishlist, "id = ?", wishlist.ID).Error; err != nil {
		c.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"message": "Produk berhasil ditambahkan ke wishlist",
			"data":    wishlist,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Produk berhasil ditambahkan ke wishlist",
		"data":    wishlist,
	})
}

func DeleteWishlistItem(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	wishlistID := strings.TrimSpace(c.Param("id"))
	if wishlistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist ID wajib diisi"})
		return
	}

	result := config.DB.Where("id = ? AND user_id = ?", wishlistID, userID).Delete(&models.Wishlist{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus wishlist"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wishlist tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Wishlist berhasil dihapus",
	})
}

func GetWishlistItems(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var wishlists []models.Wishlist
	if err := config.DB.
		Preload("Product.Supplier").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&wishlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil wishlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   wishlists,
	})
}
