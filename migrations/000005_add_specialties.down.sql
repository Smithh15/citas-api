ALTER TABLE doctor_profiles ADD COLUMN specialty VARCHAR(100);

UPDATE doctor_profiles dp
SET specialty = s.name
FROM doctor_specialties ds
JOIN specialties s ON s.id = ds.specialty_id
WHERE ds.doctor_id = dp.id;

DROP TABLE doctor_specialties;
DROP TABLE specialties;
