package lincstation

import "errors"

// Erreurs sentinelles du domaine lincstation.

// ErrInvalidColor est retournée quand une couleur invalide est fournie.
var ErrInvalidColor = errors.New("lincstation: invalid color")

// ErrInvalidLED est retournée quand un identifiant de LED invalide est utilisé.
var ErrInvalidLED = errors.New("lincstation: invalid LED")

// ErrInvalidNumber est retournée quand le numéro SATA/NVMe est hors range.
var ErrInvalidNumber = errors.New("lincstation: invalid number")

// ErrBusNotFound est retournée quand aucun bus I2C approprié n'est trouvé.
var ErrBusNotFound = errors.New("lincstation: I2C bus not found")

// ErrInvalidAnimation est retournée quand une animation inconnue est fournie.
var ErrInvalidAnimation = errors.New("lincstation: invalid animation")

// ErrInvalidLoop est retournée quand le numéro de loop n'est pas 1 ou 2.
var ErrInvalidLoop = errors.New("lincstation: invalid loop number (must be 1 or 2)")
