package lib_db

import (
	lib_db_manager "application/src/lib/db/manager"
	lib_redis "application/src/lib/redis"
	"errors"
	"gorm.io/gorm"
)

/*
: Quản lý kết nối và tương tác với cơ sở dữ liệu và Redis
db: Kết nối cơ sở dữ liệu chính (không trong giao dịch)
transaction: Kết nối cơ sở dữ liệu trong giao dịch (nếu có)
redis: Client kết nối với Redis để lưu trữ cache
modelManager: Quản lý model và cache như đã phân tích trước đó
*/
type Client struct {
	db          *gorm.DB
	transaction *gorm.DB
	redis       *lib_redis.Client

	modelManager *lib_db_manager.ModelManager
}

func NewClient(db *gorm.DB, redis *lib_redis.Client) *Client {
	return &Client{
		db:           db,
		redis:        redis,
		transaction:  nil,
		modelManager: nil,
	}
}

/*
Mục đích: Đóng kết nối và dọn dẹp tài nguyên
Hành động: Rollback giao dịch nếu đang có
Comment tiếng Nhật: "Đóng kết nối. Client.Close được gọi mỗi khi một tác vụ hoàn thành. sqlDB.Close nên được gọi khi kết thúc quá trình."
*/
func (c *Client) Close() {
	c.RollbackTransaction()
	// Client.Closeは何か作業が終わるたびに呼ばれる想定.sqlDB.Closeはプロセスの終了時に行うべき.
	// sqlDB, _ := c.db.DB()
	// if sqlDB != nil {
	// 	sqlDB.Close()
	// }
}

/*
Mục đích: Lấy kết nối DB hiện tại (trong hoặc ngoài giao dịch)
Logic: Trả về kết nối giao dịch nếu đang trong giao dịch, ngược lại trả về kết nối chính
Comment tiếng Nhật: "Lấy thông tin kết nối DB"
*/
func (c *Client) GetDB() *gorm.DB {
	if c.transaction != nil {
		return c.transaction
	}
	return c.db
}

/*
Mục đích: Lấy client Redis để thao tác với cache
Comment tiếng Nhật: "Lấy client Redis cho cache"
*/
func (c *Client) GetRedis() *lib_redis.Client {
	return c.redis
}

/*
Mục đích: Kiểm tra xem hiện tại có đang trong giao dịch hay không
Comment tiếng Nhật: "Kiểm tra có đang trong giao dịch không"
*/
func (c *Client) GetIsInTransaction() bool {
	return c.transaction != nil
}

/*
Mục đích: Quản lý vòng đời của ModelManager
Các phương thức:
DeleteModelManager(): Xóa ModelManager hiện tại (không cho phép trong giao dịch)
GetModelManager(): Lấy ModelManager hiện tại hoặc tạo mới nếu chưa có
Comment tiếng Nhật: "Xóa ModelManager" và "Lấy ModelManager"
*/
func (c *Client) DeleteModelManager() error {
	if c.GetIsInTransaction() {
		return errors.New("トランザクション中です")
	}
	c.modelManager = nil
	return nil
}
func (c *Client) GetModelManager() *lib_db_manager.ModelManager {
	if c.modelManager == nil {
		c.modelManager = lib_db_manager.NewModelManager(c.GetDB(), c.redis, c.GetIsInTransaction())
	}
	return c.modelManager
}

/*
Mục đích: Quản lý vòng đời của giao dịch cơ sở dữ liệu
Các phương thức:
StartTransaction(): Bắt đầu một giao dịch mới
CommitTransaction(): Xác nhận và kết thúc giao dịch hiện tại
RollbackTransaction(): Hủy bỏ và kết thúc giao dịch hiện tại
Xử lý đặc biệt:
Nếu cố gắng bắt đầu giao dịch khi đã có giao dịch, hệ thống sẽ panic
Khi commit giao dịch, nếu có lỗi sẽ tự động rollback
Sau khi commit thành công, gọi WriteEnd() trên ModelManager để xử lý các tác vụ sau commit
Comment tiếng Nhật: "Bắt đầu xử lý giao dịch", "Kết thúc xử lý giao dịch", và "Rollback và kết thúc xử lý giao dịch"
Cách hoạt độn
*/
func (c *Client) StartTransaction() *gorm.DB {
	if c.transaction != nil {
		// 重複.ここに来るのは設計ミス.クリティカルすぎるのでpanic.
		c.RollbackTransaction()
		panic("Transactions are duplicated.")
	}
	c.transaction = c.db.Begin()
	c.modelManager = nil
	return c.transaction
}
func (c *Client) CommitTransaction() error {
	if c.transaction == nil {
		return nil
	}
	err := c.transaction.Commit().Error
	if err != nil {
		c.RollbackTransaction()
	}
	if c.modelManager != nil {
		c.modelManager.WriteEnd()
	}
	c.transaction = nil
	c.modelManager = nil
	return err
}

func (c *Client) RollbackTransaction() error {
	if c.transaction == nil {
		return nil
	}
	err := c.transaction.Rollback().Error
	c.transaction = nil
	c.modelManager = nil
	return err
}
