CREATE TABLE
    cats (
        id INT AUTO_INCREMENT PRIMARY KEY,
        cat_name VARCHAR(50),
        years_of_experience INT NOT NULL,
        salary INT NOT NULL,
        breed VARCHAR(100) NOT NULL
    );

CREATE TABLE
    missions (
        id INT AUTO_INCREMENT PRIMARY KEY,
        cat_id INT,
        completed BOOLEAN NOT NULL DEFAULT FALSE,
        CONSTRAINT fk_mission_cat FOREIGN KEY (cat_id) REFERENCES cats (id) ON DELETE CASCADE
    );

CREATE TABLE
    targets (
        id INT AUTO_INCREMENT PRIMARY KEY,
        mission_id INT NOT NULL,
        target_name VARCHAR(50) NOT NULL,
        country VARCHAR(100) NOT NULL,
        notes TEXT,
        completed BOOLEAN NOT NULL DEFAULT FALSE,
        CONSTRAINT fk_target_mission FOREIGN KEY (mission_id) REFERENCES missions (id) ON DELETE CASCADE
    );