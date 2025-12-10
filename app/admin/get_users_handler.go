package admin

import "context"

type GetUsersRequest struct {
}

type GetUsersResponse []UserInfo

type UserInfo struct {
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Provider *string `json:"provider,omitempty"`
}

type GetUsersHandler struct {
	repository Repository
}

func NewGetUsersHandler(repository Repository) *GetUsersHandler {
	return &GetUsersHandler{
		repository: repository,
	}
}

// @Summary     Get Users
// @Description Retrieve a list of all users
// @Tags        Admin
// @Accept      json
// @Produce     json
// @Param       request body GetUsersRequest true "Get Users request"
// @Success     200 {object} GetUsersResponse
// @Failure     400
// @Failure     500
// @Router      /admin/users [post]
func (h *GetUsersHandler) Handle(ctx context.Context, req *GetUsersRequest) (*GetUsersResponse, int, error) {
	users, err := h.repository.GetAllUsers(ctx)
	if err != nil {
		return nil, 500, err
	}

	var response GetUsersResponse
	for _, user := range users {
		userInfo := UserInfo{
			Username: user.Username,
			Email:    user.Email,
			Provider: user.Provider,
		}
		response = append(response, userInfo)
	}

	return &response, 200, nil
}
