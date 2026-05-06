package resp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/pkg/errcode"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)

	if body.Code != 0 {
		t.Errorf("code = %d, want 0", body.Code)
	}
	if body.Msg != "success" {
		t.Errorf("msg = %s, want success", body.Msg)
	}
	if body.Data == nil {
		t.Error("data should not be nil")
	}
}

func TestSuccessNil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, nil)

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)

	if body.Code != 0 {
		t.Errorf("code = %d, want 0", body.Code)
	}
}

func TestFail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, errcode.ErrUserNotFound)

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)

	if body.Code != errcode.ErrUserNotFound.Code {
		t.Errorf("code = %d, want %d", body.Code, errcode.ErrUserNotFound.Code)
	}
	if body.Msg != errcode.ErrUserNotFound.Message {
		t.Errorf("msg = %s, want %s", body.Msg, errcode.ErrUserNotFound.Message)
	}
}

func TestFailWithDetail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	detailed := errcode.ErrInvalidParam.WithDetail("userId格式错误")
	Fail(c, detailed)

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)

	if body.Detail != "userId格式错误" {
		t.Errorf("detail = %s, want 'userId格式错误'", body.Detail)
	}
}

func TestFailWithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithStatus(c, http.StatusUnauthorized, errcode.ErrUnauthorized)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != errcode.ErrUnauthorized.Code {
		t.Errorf("code = %d, want %d", body.Code, errcode.ErrUnauthorized.Code)
	}
}

func TestFailWithPlainError(t *testing.T) {
	// Fail 应兼容普通 error（非 *errcode.Error）
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	plainErr := json.Unmarshal([]byte("{"), &struct{}{}) // 产生一个普通的 error
	Fail(c, plainErr)

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)

	if body.Code != 50000 {
		t.Errorf("code = %d, want 50000 (plain error should use fallback code)", body.Code)
	}
}

// --- New tests ---

func TestFailWithPlainErrorWithStatus(t *testing.T) {
	// FailWithStatus with a plain (non-errcode) error
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	plainErr := errors.New("something went wrong")
	FailWithStatus(c, http.StatusTooManyRequests, plainErr)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	var body Response
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != 50000 {
		t.Errorf("code = %d, want 50000 (plain error should use fallback code)", body.Code)
	}
	if body.Msg != "something went wrong" {
		t.Errorf("msg = %s, want 'something went wrong'", body.Msg)
	}
}

func TestFailNilError(t *testing.T) {
	// Calling Fail with nil error causes a panic (err.Error() on nil interface)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when calling Fail with nil error")
		}
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, nil)
}
