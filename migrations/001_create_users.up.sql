CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  full_name VARCHAR(150),
  is_verified BOOLEAN NOT NULL DEFAULT false,
  status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  password_changed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,

  CONSTRAINT chk_users_status
    CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED'))
);

CREATE UNIQUE INDEX users_email_active_unique_idx ON users (LOWER(email)) WHERE deleted_at IS NULL;

CREATE TABLE system_roles (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL,
  name VARCHAR(100) NOT NULL,
  description TEXT NOT NULL,
  is_system BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT uq_system_roles_code UNIQUE (code)
);

CREATE TABLE system_permissions (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(100) NOT NULL,
  name VARCHAR(150) NOT NULL,
  description TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT uq_system_permissions_code UNIQUE (code)
);

CREATE TABLE system_role_permissions (
  role_id BIGINT NOT NULL REFERENCES system_roles(id) ON DELETE CASCADE,
  permission_id BIGINT NOT NULL REFERENCES system_permissions(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_system_roles (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id BIGINT NOT NULL REFERENCES system_roles(id) ON DELETE CASCADE,
  assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_system_roles_role_id
  ON user_system_roles (role_id);

CREATE INDEX idx_user_system_roles_assigned_by
  ON user_system_roles (assigned_by)
  WHERE assigned_by IS NOT NULL;

CREATE INDEX idx_system_role_permissions_permission_id
  ON system_role_permissions (permission_id);

INSERT INTO system_roles (code, name, description)
VALUES
  (
    'ADMIN',
    'Administrator',
    'Quản trị vận hành: xem danh sách user, khoá hoặc mở user, xem workspace, xử lý abuse hoặc report; không bao gồm các quyền xoá dữ liệu cực nhạy hay cấp quyền SUPER_ADMIN.'
  ),
  (
    'SUPPORT',
    'Support',
    'Hỗ trợ CS hoặc ops: xem user và workspace ở mức read-only, reset các trạng thái an toàn, resend verification, hỗ trợ ticket; không được đổi quyền, xoá dữ liệu, hay sửa dữ liệu nhạy cảm.'
  ),
  (
    'USER',
    'User',
    'Người dùng bình thường của hệ thống.'
  );

INSERT INTO system_permissions (code, name, description)
VALUES
  ('users.read', 'View users', 'Xem danh sách và chi tiết user.'),
  ('users.suspend', 'Suspend users', 'Khoá user khỏi hệ thống.'),
  ('users.unsuspend', 'Unsuspend users', 'Mở khoá user đã bị suspend.'),
  ('workspaces.read', 'View workspaces', 'Xem danh sách và chi tiết workspace.'),
  ('abuse_reports.review', 'Review abuse reports', 'Xem và xử lý abuse hoặc report.'),
  ('users.safe_reset', 'Safe user resets', 'Reset các trạng thái an toàn cho user mà không động vào dữ liệu nhạy cảm.'),
  ('auth.resend_verification', 'Resend verification', 'Gửi lại email verification để hỗ trợ người dùng.'),
  ('support_tickets.assist', 'Assist support tickets', 'Hỗ trợ xử lý ticket vận hành hoặc CS.');

INSERT INTO system_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM system_roles r
JOIN system_permissions p
  ON p.code IN (
    'users.read',
    'users.suspend',
    'users.unsuspend',
    'workspaces.read',
    'abuse_reports.review'
  )
WHERE r.code = 'ADMIN';

INSERT INTO system_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM system_roles r
JOIN system_permissions p
  ON p.code IN (
    'users.read',
    'workspaces.read',
    'users.safe_reset',
    'auth.resend_verification',
    'support_tickets.assist'
  )
WHERE r.code = 'SUPPORT';
