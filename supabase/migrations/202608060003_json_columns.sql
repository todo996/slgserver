BEGIN;

-- lib/pq trả JSON/JSONB dưới dạng byte, tương thích với các hook AfterSet hiện có.
-- JSONB cũng ngăn dữ liệu cấu trúc không hợp lệ được ghi vào cơ sở dữ liệu.
ALTER TABLE public.tb_city_facility_1
    ALTER COLUMN facilities TYPE JSONB USING facilities::jsonb;

ALTER TABLE public.tb_general_1
    ALTER COLUMN skills DROP DEFAULT,
    ALTER COLUMN skills TYPE JSONB USING skills::jsonb,
    ALTER COLUMN skills SET DEFAULT '[0, 0, 0]'::jsonb;

ALTER TABLE public.tb_army_1
    ALTER COLUMN generals DROP DEFAULT,
    ALTER COLUMN soldiers DROP DEFAULT,
    ALTER COLUMN conscript_times DROP DEFAULT,
    ALTER COLUMN conscript_cnts DROP DEFAULT,
    ALTER COLUMN generals TYPE JSONB USING generals::jsonb,
    ALTER COLUMN soldiers TYPE JSONB USING soldiers::jsonb,
    ALTER COLUMN conscript_times TYPE JSONB USING conscript_times::jsonb,
    ALTER COLUMN conscript_cnts TYPE JSONB USING conscript_cnts::jsonb,
    ALTER COLUMN generals SET DEFAULT '[0, 0, 0]'::jsonb,
    ALTER COLUMN soldiers SET DEFAULT '[0, 0, 0]'::jsonb,
    ALTER COLUMN conscript_times SET DEFAULT '[0, 0, 0]'::jsonb,
    ALTER COLUMN conscript_cnts SET DEFAULT '[0, 0, 0]'::jsonb;

ALTER TABLE public.tb_war_report_1
    ALTER COLUMN b_a_army TYPE JSONB USING b_a_army::jsonb,
    ALTER COLUMN b_d_army TYPE JSONB USING b_d_army::jsonb,
    ALTER COLUMN e_a_army TYPE JSONB USING e_a_army::jsonb,
    ALTER COLUMN e_d_army TYPE JSONB USING e_d_army::jsonb,
    ALTER COLUMN b_a_general TYPE JSONB USING b_a_general::jsonb,
    ALTER COLUMN b_d_general TYPE JSONB USING b_d_general::jsonb,
    ALTER COLUMN e_a_general TYPE JSONB USING e_a_general::jsonb,
    ALTER COLUMN e_d_general TYPE JSONB USING e_d_general::jsonb,
    ALTER COLUMN rounds TYPE JSONB USING rounds::jsonb;

ALTER TABLE public.tb_coalition_1
    ALTER COLUMN members TYPE JSONB USING members::jsonb;

ALTER TABLE public.tb_role_attribute_1
    ALTER COLUMN pos_tags TYPE JSONB USING NULLIF(pos_tags, '')::jsonb;

ALTER TABLE public.tb_skill_1
    ALTER COLUMN belong_generals DROP DEFAULT,
    ALTER COLUMN belong_generals TYPE JSONB USING belong_generals::jsonb,
    ALTER COLUMN belong_generals SET DEFAULT '[]'::jsonb;

COMMIT;
