-- 카드 검색 인덱스 (#8). 측정 근거: mydocs/working/task_m050_8_stage1.md 3절
-- 검색은 낱말 하나로 후보 카드를 인덱스로 찾고(이름·번호·세트·도감 번호), 나머지 낱말은 후보에서만 확인한다.

-- +goose Up
-- 대소문자를 무시한 이름 부분 일치 (lower(name) LIKE '%낱말%')
CREATE INDEX cards_name_lower_trgm_idx ON cards USING gin (lower(name) gin_trgm_ops);
-- 앞의 0을 무시한 카드 번호 일치 ('25' = '025')
CREATE INDEX cards_local_number_idx ON cards ((ltrim(local_id, '0')));
-- 대소문자를 구분하는 이름 인덱스(#4)는 검색이 lower(name) 을 쓰므로 쓰이지 않는다 (#8 S1)
DROP INDEX cards_name_trgm_idx;

-- +goose Down
CREATE INDEX cards_name_trgm_idx ON cards USING gin (name gin_trgm_ops);
DROP INDEX cards_local_number_idx;
DROP INDEX cards_name_lower_trgm_idx;
