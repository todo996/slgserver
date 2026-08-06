BEGIN;

-- Client Cocos không truy cập Supabase trực tiếp. Toàn bộ dữ liệu game đi qua
-- backend Railway, vì vậy anon/authenticated không được phép đọc hoặc ghi bảng.
ALTER TABLE public.tb_user_info ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_login_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_login_last ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_role_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_map_role_city_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_map_role_build_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_city_facility_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_role_res_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_general_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_army_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_war_report_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_coalition_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_coalition_apply_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_role_attribute_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_coalition_log_1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tb_skill_1 ENABLE ROW LEVEL SECURITY;

DO $$
DECLARE
    api_role TEXT;
BEGIN
    FOREACH api_role IN ARRAY ARRAY['anon', 'authenticated']
    LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = api_role) THEN
            EXECUTE format(
                'REVOKE ALL ON TABLE '
                || 'public.tb_user_info, public.tb_login_history, public.tb_login_last, '
                || 'public.tb_role_1, public.tb_map_role_city_1, public.tb_map_role_build_1, '
                || 'public.tb_city_facility_1, public.tb_role_res_1, public.tb_general_1, '
                || 'public.tb_army_1, public.tb_war_report_1, public.tb_coalition_1, '
                || 'public.tb_coalition_apply_1, public.tb_role_attribute_1, '
                || 'public.tb_coalition_log_1, public.tb_skill_1 FROM %I',
                api_role
            );
            EXECUTE format(
                'REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM %I',
                api_role
            );
        END IF;
    END LOOP;
END;
$$;

COMMIT;
