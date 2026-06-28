package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleUser     Role = "user"
	RoleSupplier Role = "supplier"
	RoleAdmin    Role = "admin"
)

// User merepresentasikan entitas Pengguna, baik itu UMKM, Supplier maupun Admin.
type User struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	BusinessName string    `gorm:"type:varchar(255)" json:"business_name"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"` // Disembunyikan di JSON response
	Role         Role      `gorm:"type:varchar(20);not null" json:"role"`
	Address      string    `gorm:"type:text" json:"address"`
	Category     string    `gorm:"type:varchar(100)" json:"category"`                // Untuk Kategori Bahan Supplier
	Region       string    `gorm:"type:varchar(100)" json:"region"`                  // Untuk Wilayah Supply
	PICName      string    `gorm:"type:varchar(255)" json:"pic_name"`                // Penanggung jawab akun bisnis
	Phone        string    `gorm:"type:varchar(30)" json:"phone"`                    // Nomor kontak utama
	DocumentURL  string    `gorm:"type:varchar(255)" json:"document_url"`            // Opsional untuk UMKM, Wajib bagi Supplier
	Status       string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, active, suspended
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Products []Product `gorm:"foreignKey:SupplierID" json:"products,omitempty"`
	Orders   []Order   `gorm:"foreignKey:UmkmID" json:"purchased_orders,omitempty"`
	Sales    []Order   `gorm:"foreignKey:SupplierID" json:"sales_orders,omitempty"`
}

// Product merepresentasikan barang yang dijual oleh Supplier
type Product struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	SupplierID    string    `gorm:"type:varchar(36);not null;index" json:"supplier_id"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	Category      string    `gorm:"type:varchar(100)" json:"category"`
	Price         float64   `gorm:"type:numeric(15,2);not null" json:"price"`
	Stock         int       `gorm:"type:int;default:0" json:"stock"`
	Description   string    `gorm:"type:text" json:"description"`
	Location      string    `gorm:"type:varchar(255)" json:"location"`
	ImageURL      string    `gorm:"type:varchar(255)" json:"image_url"`
	RatingAverage float64   `gorm:"-" json:"rating_average"`
	ReviewCount   int       `gorm:"-" json:"review_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Supplier User `gorm:"foreignKey:SupplierID;references:ID" json:"supplier,omitempty"`
}

// OrderStatus defines the lifecycle of an order
type OrderStatus string

const (
	OrderPending                     OrderStatus = "pending"
	OrderPendingSupplierConfirmation OrderStatus = "pending_supplier_confirmation"
	OrderRejectedBySupplier          OrderStatus = "rejected_by_supplier"
	OrderStockUnavailable            OrderStatus = "stock_unavailable"
	OrderSupplierConfirmed           OrderStatus = "supplier_confirmed"
	OrderPaymentPending              OrderStatus = "payment_pending"
	OrderPaymentRequestFailed        OrderStatus = "payment_request_failed"
	OrderPaid                        OrderStatus = "paid"
	OrderPaymentFailed               OrderStatus = "payment_failed"
	OrderShipmentCreated             OrderStatus = "shipment_created"
	OrderProcessing                  OrderStatus = "processing"
	OrderShipped                     OrderStatus = "shipped"
	OrderCompleted                   OrderStatus = "completed"
	OrderCancelled                   OrderStatus = "cancelled"
)

// Order merepresentasikan tagihan pembelian antara UMKM dan Supplier
type Order struct {
	ID             string      `gorm:"type:varchar(36);primaryKey" json:"id"`
	UmkmID         string      `gorm:"type:varchar(36);not null;index;index:idx_orders_umkm_created,priority:1" json:"umkm_id"`
	SupplierID     string      `gorm:"type:varchar(36);not null;index;index:idx_orders_supplier_created,priority:1" json:"supplier_id"`
	ProductID      string      `gorm:"type:varchar(36);not null;index" json:"product_id"`
	Quantity       int         `gorm:"not null" json:"quantity"`
	TotalBasePrice float64     `gorm:"type:numeric(15,2);not null" json:"total_base_price"`
	SystemFee      float64     `gorm:"type:numeric(15,2);default:0" json:"system_fee"`
	GrandTotal     float64     `gorm:"type:numeric(15,2);not null" json:"grand_total"`
	Status         OrderStatus `gorm:"type:varchar(40);default:'pending_supplier_confirmation';index" json:"status"`
	StockDeducted  bool        `gorm:"default:false" json:"stock_deducted"`
	IsReviewed     bool        `gorm:"-" json:"is_reviewed"`
	CreatedAt      time.Time   `gorm:"index:idx_orders_supplier_created,priority:2;index:idx_orders_umkm_created,priority:2" json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`

	// Relations
	Product  Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Umkm     User     `gorm:"foreignKey:UmkmID" json:"umkm,omitempty"`
	Supplier User     `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	Payment  *Payment `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
}

type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending"
	PaymentSuccess PaymentStatus = "success"
	PaymentFailed  PaymentStatus = "failed"
)

// Payment menyimpan jejak request pembayaran ke ekosistem SmartBank.
type Payment struct {
	ID               string        `gorm:"type:varchar(36);primaryKey" json:"id"`
	OrderID          string        `gorm:"type:varchar(36);not null;index" json:"order_id"`
	UserID           string        `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Amount           float64       `gorm:"type:numeric(15,2);not null" json:"amount"`
	SupplierFee      float64       `gorm:"type:numeric(15,2);default:0" json:"supplier_fee"`
	Status           PaymentStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentMethod    string        `gorm:"type:varchar(50)" json:"payment_method"`
	VirtualAccount   string        `gorm:"type:varchar(100)" json:"virtual_account"`
	PaymentReference string        `gorm:"type:varchar(100);index" json:"payment_reference"`
	ExternalOrderID  string        `gorm:"type:varchar(100);index" json:"external_order_id"`
	GatewayStatus    string        `gorm:"type:varchar(50)" json:"gateway_status"`
	CallbackStatus   string        `gorm:"type:varchar(50)" json:"callback_status"`
	GatewayResponse  string        `gorm:"type:text" json:"gateway_response,omitempty"`
	PaidAt           *time.Time    `json:"paid_at,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`

	Order Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// FinanceLog menyimpan ledger transaksi SupplierHub untuk audit biaya layanan.
type FinanceLog struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	OrderID         string    `gorm:"type:varchar(36);index" json:"order_id"`
	PaymentID       *string   `gorm:"type:varchar(36);index" json:"payment_id,omitempty"`
	UmkmID          string    `gorm:"type:varchar(36);index" json:"umkm_id"`
	SupplierID      string    `gorm:"type:varchar(36);index" json:"supplier_id"`
	ProductID       string    `gorm:"type:varchar(36);index" json:"product_id"`
	Subtotal        float64   `gorm:"type:numeric(15,2);default:0" json:"subtotal"`
	SupplierFee     float64   `gorm:"type:numeric(15,2);default:0" json:"supplier_fee"`
	GrandTotal      float64   `gorm:"type:numeric(15,2);default:0" json:"grand_total"`
	PaymentStatus   string    `gorm:"type:varchar(30)" json:"payment_status"`
	OrderStatus     string    `gorm:"type:varchar(40)" json:"order_status"`
	TransactionType string    `gorm:"type:varchar(50);index" json:"transaction_type"`
	Note            string    `gorm:"type:text" json:"note"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Order   Order    `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Payment *Payment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
}

// RequestLog mencatat request penting untuk kebutuhan audit integrasi.
type RequestLog struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID          string    `gorm:"type:varchar(36);index" json:"user_id,omitempty"`
	Role            string    `gorm:"type:varchar(30)" json:"role,omitempty"`
	Method          string    `gorm:"type:varchar(10)" json:"method"`
	Path            string    `gorm:"type:varchar(255);index" json:"path"`
	StatusCode      int       `json:"status_code"`
	IPAddress       string    `gorm:"type:varchar(100)" json:"ip_address"`
	UserAgent       string    `gorm:"type:text" json:"user_agent"`
	RequestBody     string    `gorm:"type:text" json:"request_body,omitempty"`
	ResponseMessage string    `gorm:"type:text" json:"response_message,omitempty"`
	LatencyMS       int64     `json:"latency_ms"`
	CreatedAt       time.Time `json:"created_at"`
}

// ShipmentLog menyimpan hasil request pengiriman ke LogistiKita.
type ShipmentLog struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	OrderID         string    `gorm:"type:varchar(36);index" json:"order_id"`
	SupplierID      string    `gorm:"type:varchar(36);index" json:"supplier_id"`
	UmkmID          string    `gorm:"type:varchar(36);index" json:"umkm_id"`
	ProductID       string    `gorm:"type:varchar(36);index" json:"product_id"`
	ShipmentID      string    `gorm:"type:varchar(100);index" json:"shipment_id"`
	Status          string    `gorm:"type:varchar(50)" json:"status"`
	GatewayResponse string    `gorm:"type:text" json:"gateway_response,omitempty"`
	ErrorMessage    string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Order Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// Notification menyimpan pesan sistem untuk user tertentu.
type Notification struct {
	ID         string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID     string     `gorm:"type:varchar(36);not null;index;index:idx_notifications_user_role_created,priority:1" json:"user_id"`
	Role       string     `gorm:"type:varchar(30);index;index:idx_notifications_user_role_created,priority:2" json:"role"`
	Title      string     `gorm:"type:varchar(150);not null" json:"title"`
	Message    string     `gorm:"type:text;not null" json:"message"`
	Type       string     `gorm:"type:varchar(50);index" json:"type"`
	SourceType string     `gorm:"type:varchar(50)" json:"source_type"`
	SourceID   string     `gorm:"type:varchar(36);index" json:"source_id"`
	IsRead     bool       `gorm:"default:false;index" json:"is_read"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `gorm:"index:idx_notifications_user_role_created,priority:3" json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// ChatConversation menyimpan kanal percakapan satu UMKM dengan satu supplier.
type ChatConversation struct {
	ID            string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UmkmID        string     `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_chat_conversation_pair" json:"umkm_id"`
	SupplierID    string     `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_chat_conversation_pair" json:"supplier_id"`
	LastMessage   string     `gorm:"type:text" json:"last_message"`
	LastSenderID  string     `gorm:"type:varchar(36);index" json:"last_sender_id"`
	LastMessageAt *time.Time `gorm:"index" json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Umkm     User          `gorm:"foreignKey:UmkmID;references:ID" json:"umkm,omitempty"`
	Supplier User          `gorm:"foreignKey:SupplierID;references:ID" json:"supplier,omitempty"`
	Messages []ChatMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

// ChatMessage menyimpan isi pesan antar UMKM dan supplier.
type ChatMessage struct {
	ID             string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ConversationID string     `gorm:"type:varchar(36);not null;index" json:"conversation_id"`
	SenderID       string     `gorm:"type:varchar(36);not null;index" json:"sender_id"`
	ReceiverID     string     `gorm:"type:varchar(36);not null;index" json:"receiver_id"`
	Message        string     `gorm:"type:text;not null" json:"message"`
	IsRead         bool       `gorm:"default:false;index" json:"is_read"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Conversation ChatConversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
	Sender       User             `gorm:"foreignKey:SenderID;references:ID" json:"sender,omitempty"`
	Receiver     User             `gorm:"foreignKey:ReceiverID;references:ID" json:"receiver,omitempty"`
}

// Wishlist menyimpan produk bahan baku favorit milik UMKM di database.
type Wishlist struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID      string    `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_wishlist_user_product" json:"user_id"`
	BahanBakuID string    `gorm:"column:bahan_baku_id;type:varchar(36);not null;index;uniqueIndex:idx_wishlist_user_product" json:"bahan_baku_id"`
	CreatedAt   time.Time `json:"created_at"`

	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Product Product `gorm:"foreignKey:BahanBakuID;references:ID" json:"product,omitempty"`
}

// Log merepresentasikan rekam jejak aktivitas (Audit) untuk Admin
type Log struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"type:varchar(36);index" json:"user_id"` // Siapa yang melakukan aksi
	Action      string    `gorm:"type:varchar(100);not null" json:"action"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// BeforeCreate hooks untuk generate UUID secara otomatis sebelum insert ke database
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}

func (p *Product) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return
}

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return
}

func (p *Payment) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return
}

func (f *FinanceLog) BeforeCreate(tx *gorm.DB) (err error) {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return
}

func (r *RequestLog) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}

func (s *ShipmentLog) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return
}

func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return
}

func (c *ChatConversation) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return
}

func (w *Wishlist) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return
}

// Review menyimpan penilaian bintang dan komentar dari UMKM untuk produk tertentu
type Review struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	OrderID   string    `gorm:"type:varchar(36);not null;index" json:"order_id"`
	ProductID string    `gorm:"type:varchar(36);not null;index" json:"product_id"`
	UmkmID    string    `gorm:"type:varchar(36);not null;index" json:"umkm_id"`
	Rating    int       `gorm:"not null" json:"rating"` // 1 - 5
	Comment   string    `gorm:"type:text" json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Umkm User `gorm:"foreignKey:UmkmID;references:ID" json:"umkm,omitempty"`
}

func (r *Review) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}
