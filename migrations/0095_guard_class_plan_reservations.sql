-- +goose Up
-- +goose StatementBegin
-- Only students explicitly reserved by an admin-approved planning workflow
-- are protected. Unreserved legacy onboarding/class operations are unchanged.
CREATE TRIGGER IF NOT EXISTS trg_planning_reserved_santri_class_change
BEFORE UPDATE OF kelas_id ON santri
FOR EACH ROW
WHEN COALESCE(NEW.kelas_id,-1)<>COALESCE(OLD.kelas_id,-1)
 AND EXISTS (
  SELECT 1 FROM class_plan_students ps
  JOIN class_plans p ON p.id=ps.plan_id
  WHERE ps.santri_id=OLD.id AND ps.status='reserved' AND p.status='pending_admin'
 )
BEGIN
 SELECT RAISE(ABORT,'santri reserved by approved class plan');
END;

CREATE TRIGGER IF NOT EXISTS trg_planning_reserved_santri_delete
BEFORE DELETE ON santri
FOR EACH ROW
WHEN EXISTS (
  SELECT 1 FROM class_plan_students ps
  JOIN class_plans p ON p.id=ps.plan_id
  WHERE ps.santri_id=OLD.id AND ps.status='reserved' AND p.status='pending_admin'
 )
BEGIN
 SELECT RAISE(ABORT,'santri reserved by approved class plan');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_planning_reserved_santri_delete;
DROP TRIGGER IF EXISTS trg_planning_reserved_santri_class_change;
-- +goose StatementEnd
