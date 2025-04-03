package lib_db_models

/*
Mục đích: Định nghĩa một giao diện chung cho tất cả các model trong hệ thống
Các phương thức:
GetIsMaster(): Kiểm tra xem model có phải là dữ liệu master hay không
GetIsForUpdate(): Kiểm tra xem model có đang được đánh dấu để cập nhật hay không
SetIsForUpdate(): Đặt trạng thái cập nhật cho model
*/
type ModelInterface interface {
	GetIsMaster() bool
	GetIsForUpdate() bool
	SetIsForUpdate(bool)
}

type BaseModel struct {
	forUpdate bool
}

func (bm BaseModel) GetIsMaster() bool {
	return false
}
func (bm *BaseModel) GetIsForUpdate() bool {
	return bm.forUpdate
}
func (bm *BaseModel) SetIsForUpdate(flag bool) {
	bm.forUpdate = flag
}

type BaseMaster struct {
	forUpdate bool
}

func (bm BaseMaster) GetIsMaster() bool {
	return true
}
func (bm *BaseMaster) GetIsForUpdate() bool {
	return bm.forUpdate
}
func (bm *BaseMaster) SetIsForUpdate(flag bool) {
	bm.forUpdate = flag
}
