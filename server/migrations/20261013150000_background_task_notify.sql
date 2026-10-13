-- 调度通知与任务写入属于同一事务，仅在提交后交付；载荷仅包含 Schema 与消费者。
CREATE FUNCTION notify_background_task_schedule() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    consumer text;
BEGIN
    IF TG_TABLE_NAME = 'background_tasks' THEN
        IF TG_OP = 'UPDATE' THEN
            IF ROW(NEW.kind, NEW.status, NEW.next_run_at, NEW.lease_expires_at)
                IS NOT DISTINCT FROM ROW(OLD.kind, OLD.status, OLD.next_run_at, OLD.lease_expires_at) THEN
                RETURN NEW;
            END IF;
        END IF;
        consumer := CASE NEW.kind
            WHEN 'DINGTALK_NOTIFICATION' THEN 'notification'
            WHEN 'OBJECT_STORAGE_DELETION' THEN 'object_deletion'
            WHEN 'DINGTALK_APPROVAL_CREATE' THEN 'dingtalk_approval'
            WHEN 'DINGTALK_DIRECTORY_SYNC' THEN 'dingtalk_directory'
            ELSE NULL
        END;
    ELSE
        IF TG_OP = 'UPDATE' THEN
            IF ROW(NEW.status, NEW.next_run_at, NEW.processing_expires_at)
                IS NOT DISTINCT FROM ROW(OLD.status, OLD.next_run_at, OLD.processing_expires_at) THEN
                RETURN NEW;
            END IF;
        END IF;
        consumer := 'dingtalk_approval';
    END IF;
    IF consumer IS NOT NULL THEN
        PERFORM pg_notify('roncin_task_schedule_changed',
            json_build_object('schema', TG_TABLE_SCHEMA, 'consumer', consumer)::text);
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER background_tasks_schedule_changed
AFTER INSERT OR UPDATE OF kind, status, next_run_at, lease_expires_at ON background_tasks
FOR EACH ROW EXECUTE FUNCTION notify_background_task_schedule();

CREATE TRIGGER dingtalk_approval_inbox_schedule_changed
AFTER INSERT OR UPDATE OF status, next_run_at, processing_expires_at ON ding_talk_approval_inbox_events
FOR EACH ROW EXECUTE FUNCTION notify_background_task_schedule();
