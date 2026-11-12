# Anbieter

![Screenshot von Anbieter auf Deutsch.](../assets/de/providers.png)

## Was diese Seite tut

Listet KI-Anbieter, Konten, Verbindungsstatus und verfügbare Modelle mit Raster-/Listenansicht und Verbindungstests.

## Wann verwenden

Verwenden Sie diese Seite, um diesen Teil des Gateways zu prüfen, zu konfigurieren oder zu betreiben.

## Schritt für Schritt

1. Öffnen Sie **Anbieter** über die Seitenleiste.
2. Prüfen Sie die Kennzahlen und die Haupttabelle.
3. Verwenden Sie die unten beschriebenen Aktionen; Änderungen gelten sofort.
4. Nutzen Sie Raster/Liste, filtern Sie nach Authentifizierung oder Fähigkeit und testen Sie den Anbieter vor der Aktivierung.

## Felder und Aktionen

| Feld oder Aktion | Auf dieser Seite sichtbare Bezeichnung oder Aktion. |
|---|---|
| + Anbieter hinzufügen | Auf dieser Seite sichtbare Bezeichnung oder Aktion. |
| Raster | Auf dieser Seite sichtbare Bezeichnung oder Aktion. |
| Liste | Auf dieser Seite sichtbare Bezeichnung oder Aktion. |
| Testen | Auf dieser Seite sichtbare Bezeichnung oder Aktion. |
| Status | Auf dieser Seite sichtbare Bezeichnung oder Aktion. |

## Häufige Fehler und Zustände

- Wenn Daten nicht geladen werden, versuchen Sie es erneut. Prüfen Sie Gateway und Anbieter, wenn der Fehler bleibt.
- Wenn eine Aktion deaktiviert ist, prüfen Sie Rolle und Tenant-Bereich.
- Leere Zustände sind vor der ersten Anfrage normal; folgen Sie der Handlungsaufforderung.

## Proxy pro Konto

Konfiguriere Webshare unter **Proxy-Netzwerk**, teste den API-Key und synchronisiere die Direct-Proxy-Liste. Im Provider-Konto wähle Off, Preferred oder Required; zugewiesener Endpoint, Region und Gesundheit erscheinen im selben Block. Ein Proxy in Quarantäne oder ohne Zuweisung fällt bei Preferred auf direkt zurück und blockiert den Aufruf bei Required.

## Konten und Tests

Die Anbieterdetailseite behält die konsolidierte Zusammenfassung und ergänzt eine Zusammenfassung pro Konto mit Aktivstatus, Gesundheit, verfügbaren/nutzbaren Modellen und Fähigkeiten. Wähle über **Testkonto** oberhalb der Modellliste das Konto für den Modelltest. Nur aktive Konten mit nutzbaren Modellen sind auswählbar, die letzte gültige Auswahl wird gespeichert und das Ergebnis nennt immer das verwendete Konto. Jedes Konto hat außerdem einen eigenen Testbutton; ein Fehler eines Kontos beeinflusst die anderen nicht.

## Verwandte Leitfäden

- [Endpunkte](endpoints.md)
- [Übersicht](dashboard.md)
- [Kosten](cost.md)

> Screenshot von Anbieter auf Deutsch.
