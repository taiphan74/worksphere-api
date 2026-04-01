package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"worksphere-api/internal/workspace/dto"
)

// ── WorkspaceService Mock ──

type MockWorkspaceService struct {
	mock.Mock
}

func (m *MockWorkspaceService) CreateWorkspace(ctx context.Context, userID uuid.UUID, req dto.CreateWorkspaceRequest) (dto.WorkspaceResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(dto.WorkspaceResponse), args.Error(1)
}

func (m *MockWorkspaceService) GetWorkspaceByID(ctx context.Context, userID, id uuid.UUID) (dto.WorkspaceResponse, error) {
	args := m.Called(ctx, userID, id)
	return args.Get(0).(dto.WorkspaceResponse), args.Error(1)
}

func (m *MockWorkspaceService) GetWorkspaceBySlug(ctx context.Context, userID uuid.UUID, slug string) (dto.WorkspaceResponse, error) {
	args := m.Called(ctx, userID, slug)
	return args.Get(0).(dto.WorkspaceResponse), args.Error(1)
}

func (m *MockWorkspaceService) ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]dto.WorkspaceResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]dto.WorkspaceResponse), args.Error(1)
}

func (m *MockWorkspaceService) UpdateWorkspace(ctx context.Context, userID, id uuid.UUID, req dto.UpdateWorkspaceRequest) (dto.WorkspaceResponse, error) {
	args := m.Called(ctx, userID, id, req)
	return args.Get(0).(dto.WorkspaceResponse), args.Error(1)
}

func (m *MockWorkspaceService) DeleteWorkspace(ctx context.Context, userID, id uuid.UUID) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

// ── InvitationService Mock ──

type MockInvitationService struct {
	mock.Mock
}

func (m *MockInvitationService) SendInvitation(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.SendInvitationRequest) (dto.InvitationResponse, error) {
	args := m.Called(ctx, requesterID, workspaceID, req)
	return args.Get(0).(dto.InvitationResponse), args.Error(1)
}

func (m *MockInvitationService) GetInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) (dto.InvitationResponse, error) {
	args := m.Called(ctx, requesterID, invitationID)
	return args.Get(0).(dto.InvitationResponse), args.Error(1)
}

func (m *MockInvitationService) ListInvitations(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.InvitationResponse, error) {
	args := m.Called(ctx, requesterID, workspaceID)
	return args.Get(0).([]dto.InvitationResponse), args.Error(1)
}

func (m *MockInvitationService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	args := m.Called(ctx, token, userID)
	return args.Error(0)
}

func (m *MockInvitationService) DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	args := m.Called(ctx, token, userID)
	return args.Error(0)
}

func (m *MockInvitationService) CancelInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error {
	args := m.Called(ctx, requesterID, invitationID)
	return args.Error(0)
}

func (m *MockInvitationService) ResendInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error {
	args := m.Called(ctx, requesterID, invitationID)
	return args.Error(0)
}

// ── MemberService Mock ──

type MockMemberService struct {
	mock.Mock
}

func (m *MockMemberService) AddMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.AddMemberRequest) (dto.MemberResponse, error) {
	args := m.Called(ctx, requesterID, workspaceID, req)
	return args.Get(0).(dto.MemberResponse), args.Error(1)
}

func (m *MockMemberService) ListMembers(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.MemberResponse, error) {
	args := m.Called(ctx, requesterID, workspaceID)
	return args.Get(0).([]dto.MemberResponse), args.Error(1)
}

func (m *MockMemberService) GetMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID) (dto.MemberResponse, error) {
	args := m.Called(ctx, requesterID, workspaceID, targetUserID)
	return args.Get(0).(dto.MemberResponse), args.Error(1)
}

func (m *MockMemberService) UpdateMemberRole(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID, req dto.UpdateMemberRoleRequest) (dto.MemberResponse, error) {
	args := m.Called(ctx, requesterID, workspaceID, targetUserID, req)
	return args.Get(0).(dto.MemberResponse), args.Error(1)
}

func (m *MockMemberService) RemoveMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID) error {
	args := m.Called(ctx, requesterID, workspaceID, targetUserID)
	return args.Error(0)
}
