-- Second reviewed batch of recipe-level shopping concepts observed in Gestor packs.
-- These remain independent from brands, package sizes and supermarket SKUs.
-- Cooked dry staples (pasta/quinoa/couscous/rice) are intentionally excluded here
-- because cooked recipe quantities cannot be quoted 1:1 against dry purchase quantities.
INSERT INTO canonical_ingredients (id, name, category, subtype, default_unit) VALUES
    ('bebida_avena_sin_azucar', 'Bebida de avena sin azúcar', 'bebida_vegetal', 'avena_sin_azucar', 'ml'),
    ('bebida_almendra_sin_azucar', 'Bebida de almendras sin azúcar', 'bebida_vegetal', 'almendra_sin_azucar', 'ml'),
    ('jamon_cocido', 'Jamón cocido', 'carne_procesada', 'jamon_cocido', 'g'),
    ('copos_avena', 'Copos de avena', 'cereal', 'avena_copos', 'g'),
    ('muesli_sin_azucar', 'Muesli sin azúcar', 'cereal', 'muesli_sin_azucar', 'g'),
    ('atun_al_natural', 'Atún al natural', 'pescado_conserva', 'atun_al_natural', 'g'),
    ('caballa_al_natural', 'Caballa al natural', 'pescado_conserva', 'caballa_al_natural', 'g'),
    ('fiambre_pavo', 'Fiambre de pavo', 'carne_procesada', 'fiambre_pavo', 'g'),
    ('canonigos', 'Canónigos', 'verdura', 'canonigos', 'g'),
    ('tomate_cherry', 'Tomate cherry', 'verdura', 'tomate_cherry', 'g'),
    ('pimiento_rojo', 'Pimiento rojo', 'verdura', 'pimiento_rojo', 'g'),
    ('cebolla_morada', 'Cebolla morada', 'verdura', 'cebolla_morada', 'g'),
    ('mozzarella_fresca', 'Mozzarella fresca', 'lacteo', 'mozzarella_fresca', 'g'),
    ('queso_parmesano', 'Queso parmesano', 'lacteo', 'parmesano', 'g'),
    ('salsa_pesto', 'Salsa pesto', 'salsa', 'pesto', 'g'),
    ('vinagre_jerez', 'Vinagre de Jerez', 'vinagre', 'jerez', 'ml'),
    ('vinagre_modena', 'Vinagre de Módena', 'vinagre', 'modena', 'ml'),
    ('mostaza_sin_azucar', 'Mostaza sin azúcar', 'salsa', 'mostaza_sin_azucar', 'g'),
    ('nueces', 'Nueces', 'fruto_seco', 'nuez', 'g'),
    ('almendras_naturales', 'Almendras naturales', 'fruto_seco', 'almendra_natural', 'g'),
    ('semillas_calabaza', 'Semillas de calabaza', 'semilla', 'calabaza', 'g'),
    ('cebolla_crujiente', 'Cebolla crujiente', 'condimento', 'cebolla_crujiente', 'g'),
    ('tomate_seco', 'Tomate seco', 'verdura_procesada', 'tomate_seco', 'g')
ON CONFLICT (id) DO NOTHING;
