CREATE TABLE specialties (
    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name  VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE doctor_specialties (
    doctor_id     UUID NOT NULL REFERENCES doctor_profiles(id) ON DELETE CASCADE,
    specialty_id  UUID NOT NULL REFERENCES specialties(id) ON DELETE CASCADE,
    PRIMARY KEY (doctor_id, specialty_id)
);

-- Migra los datos existentes: cada specialty de texto se vuelve una fila,
-- y cada doctor queda vinculado a la suya.
INSERT INTO specialties (name)
SELECT DISTINCT specialty FROM doctor_profiles WHERE specialty IS NOT NULL;

INSERT INTO doctor_specialties (doctor_id, specialty_id)
SELECT dp.id, s.id
FROM doctor_profiles dp
JOIN specialties s ON s.name = dp.specialty;

ALTER TABLE doctor_profiles DROP COLUMN specialty;
