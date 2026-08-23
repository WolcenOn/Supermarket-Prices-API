package canonical

import (
    "strings"
    "unicode"
)

var accentReplacer = strings.NewReplacer(
    "á", "a", "à", "a", "ä", "a", "â", "a",
    "é", "e", "è", "e", "ë", "e", "ê", "e",
    "í", "i", "ì", "i", "ï", "i", "î", "i",
    "ó", "o", "ò", "o", "ö", "o", "ô", "o",
    "ú", "u", "ù", "u", "ü", "u", "û", "u",
    "ñ", "n",
)

// NormalizeText produces the stable comparison form used by canonical names
// and aliases. It deliberately removes punctuation and accents but does not
// remove semantic words such as "entera", "integral" or "redondo".
func NormalizeText(value string) string {
    value = strings.ToLower(strings.TrimSpace(value))
    value = accentReplacer.Replace(value)

    var b strings.Builder
    b.Grow(len(value))
    for _, r := range value {
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            b.WriteRune(r)
            continue
        }
        b.WriteByte(' ')
    }
    return strings.Join(strings.Fields(b.String()), " ")
}
