## Use with Uptime Kuma (HTTP JSON-Query)

Port 8080

### Überprüfen, ob ramUsagePercent größer als 80% ist:
`$boolean(ramUsagePercent > 80)`

### Überprüfen, ob swapUsagePercent größer als 50% ist:
`$boolean(swapUsagePercent > 50)`

### Überprüfen, ob diskUsagePercent größer als 80% ist:
`$boolean(diskUsagePercent > 80)`

### Überprüfen, ob load1 größer als 4 (50% Auslastung bei 8 Kernen) ist:
`$boolean(load1 > 4)`