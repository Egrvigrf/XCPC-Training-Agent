CREATE TABLE users
(
    id        VARCHAR(32)  NOT NULL,
    name      VARCHAR(255) NOT NULL,
    password  VARCHAR(255) NOT NULL,
    status    TINYINT      NOT NULL,
    is_system TINYINT      NOT NULL,

    cf_handle VARCHAR(64) NULL,
    ac_handle VARCHAR(64) NULL,

    create_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_at TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (id),
    UNIQUE KEY uk_cf_handle (cf_handle),
    UNIQUE KEY uk_ac_handle (ac_handle)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE daily_training_stats
(
    student_id       VARCHAR(32) NOT NULL,
    stat_date        DATE        NOT NULL,

    -- Codeforces 当日新增
    cf_new_total     INT         NOT NULL DEFAULT 0,
    cf_new_800       INT         NOT NULL DEFAULT 0,
    cf_new_900       INT         NOT NULL DEFAULT 0,
    cf_new_1000      INT         NOT NULL DEFAULT 0,
    cf_new_1100      INT         NOT NULL DEFAULT 0,
    cf_new_1200      INT         NOT NULL DEFAULT 0,
    cf_new_1300      INT         NOT NULL DEFAULT 0,
    cf_new_1400      INT         NOT NULL DEFAULT 0,
    cf_new_1500      INT         NOT NULL DEFAULT 0,
    cf_new_1600      INT         NOT NULL DEFAULT 0,
    cf_new_1700      INT         NOT NULL DEFAULT 0,
    cf_new_1800      INT         NOT NULL DEFAULT 0,
    cf_new_1900      INT         NOT NULL DEFAULT 0,
    cf_new_2000      INT         NOT NULL DEFAULT 0,
    cf_new_2100      INT         NOT NULL DEFAULT 0,
    cf_new_2200      INT         NOT NULL DEFAULT 0,
    cf_new_2300      INT         NOT NULL DEFAULT 0,
    cf_new_2400      INT         NOT NULL DEFAULT 0,
    cf_new_2500      INT         NOT NULL DEFAULT 0,
    cf_new_2600      INT         NOT NULL DEFAULT 0,
    cf_new_2700      INT         NOT NULL DEFAULT 0,
    cf_new_2800_plus INT         NOT NULL DEFAULT 0,

    -- AtCoder 当日新增
    ac_new_total     INT         NOT NULL DEFAULT 0,
    ac_new_0_399     INT         NOT NULL DEFAULT 0,
    ac_new_400_799   INT         NOT NULL DEFAULT 0,
    ac_new_800_1199  INT         NOT NULL DEFAULT 0,
    ac_new_1200_1599 INT         NOT NULL DEFAULT 0,
    ac_new_1600_1999 INT         NOT NULL DEFAULT 0,
    ac_new_2000_2399 INT         NOT NULL DEFAULT 0,
    ac_new_2400_2799 INT         NOT NULL DEFAULT 0,
    ac_new_2800_plus INT         NOT NULL DEFAULT 0,

    created_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (student_id, stat_date),

    CONSTRAINT fk_daily_user
        FOREIGN KEY (student_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    INDEX            idx_stat_date (stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE contest_records
(
    student_id    VARCHAR(32)  NOT NULL,
    platform      ENUM('CF','AC') NOT NULL,

    contest_id    VARCHAR(64)  NOT NULL,
    contest_name  VARCHAR(255) NOT NULL,
    contest_date  DATETIME     NOT NULL,

    contest_rank  INT          NOT NULL,
    old_rating    INT          NOT NULL,
    new_rating    INT          NOT NULL,
    rating_change INT          NOT NULL,

    performance   INT NULL,

    created_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (student_id, platform, contest_id),

    CONSTRAINT fk_contest_user
        FOREIGN KEY (student_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    INDEX         idx_contest_date (contest_date),
    INDEX         idx_platform (platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS user_ability_snapshot;

CREATE TABLE user_ability_snapshot
(
    student_id       VARCHAR(32) NOT NULL,
    stat_date        DATE        NOT NULL,

    avg_difficulty_30d DOUBLE NOT NULL DEFAULT 0,
    total_count_30d  INT         NOT NULL DEFAULT 0,
    contest_mean_5 DOUBLE NOT NULL DEFAULT 0,
    contest_cnt_180d INT         NOT NULL DEFAULT 0,

    z_avg_diff DOUBLE NOT NULL DEFAULT 0,
    z_total_count DOUBLE NOT NULL DEFAULT 0,
    z_contest DOUBLE NOT NULL DEFAULT 0,

    score_raw DOUBLE NOT NULL DEFAULT 0,
    score_100 DOUBLE NOT NULL DEFAULT 50,

    created_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (student_id, stat_date),
    INDEX            idx_stat_date (stat_date),
    CONSTRAINT fk_snap_user FOREIGN KEY (student_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS team_metric_cache
(
    stat_date
    DATE
    NOT
    NULL,
    metric
    VARCHAR
(
    32
) NOT NULL,

    n INT NOT NULL DEFAULT 0,
    mean DOUBLE NOT NULL DEFAULT 0,
    std DOUBLE NOT NULL DEFAULT 1,

    p05 DOUBLE NOT NULL DEFAULT 0,
    p95 DOUBLE NOT NULL DEFAULT 0,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY
(
    stat_date,
    metric
),
    INDEX idx_stat_date
(
    stat_date
)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;