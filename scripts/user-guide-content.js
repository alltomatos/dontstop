const SHARED = {
  en: {
    whatTitle: 'What it does',
    whenTitle: 'When to use',
    stepsTitle: 'Step by step',
    fieldsTitle: 'Fields and actions',
    errorsTitle: 'Common errors and states',
    relatedTitle: 'Related guides',
    fieldDescription: 'Label or action visible on this screen.',
    stepOpen: 'Open **{page}** from the sidebar.',
    stepReview: 'Review the indicators and the main table.',
    stepAct: 'Use the actions described below; changes take effect immediately.',
    errorLoad: 'If data fails to load, retry. If the error persists, check the gateway and provider status.',
    errorPermission: 'If an action is disabled, confirm your role and the tenant scope.',
    errorEmpty: 'Empty states are expected before the first request; follow the call to action.',
    imageAlt: 'Screenshot of the {page} screen in {locale}.'
  },
  'pt-BR': {
    whatTitle: 'O que faz',
    whenTitle: 'Quando usar',
    stepsTitle: 'Passo a passo',
    fieldsTitle: 'Campos e ações',
    errorsTitle: 'Erros e estados comuns',
    relatedTitle: 'Guias relacionados',
    fieldDescription: 'Rótulo ou ação visível nesta tela.',
    stepOpen: 'Abra **{page}** pela barra lateral.',
    stepReview: 'Revise os indicadores e a tabela principal.',
    stepAct: 'Use as ações descritas abaixo; as mudanças valem imediatamente.',
    errorLoad: 'Se os dados não carregarem, tente novamente. Se persistir, verifique o gateway e os provedores.',
    errorPermission: 'Se uma ação estiver desabilitada, confirme seu papel e o escopo do tenant.',
    errorEmpty: 'Estados vazios são esperados antes da primeira requisição; siga a chamada para ação.',
    imageAlt: 'Captura de tela de {page} em {locale}.'
  },
  'pt-PT': {
    whatTitle: 'O que faz',
    whenTitle: 'Quando utilizar',
    stepsTitle: 'Passo a passo',
    fieldsTitle: 'Campos e ações',
    errorsTitle: 'Erros e estados comuns',
    relatedTitle: 'Guias relacionados',
    fieldDescription: 'Rótulo ou ação visível neste ecrã.',
    stepOpen: 'Abra **{page}** na barra lateral.',
    stepReview: 'Reveja os indicadores e a tabela principal.',
    stepAct: 'Utilize as ações descritas abaixo; as alterações aplicam-se imediatamente.',
    errorLoad: 'Se os dados não carregarem, tente novamente. Se persistir, verifique o gateway e os fornecedores.',
    errorPermission: 'Se uma ação estiver desativada, confirme o seu papel e o âmbito do tenant.',
    errorEmpty: 'Estados vazios são esperados antes do primeiro pedido; siga a chamada para ação.',
    imageAlt: 'Captura de ecrã de {page} em {locale}.'
  },
  es: {
    whatTitle: 'Qué hace',
    whenTitle: 'Cuándo usarla',
    stepsTitle: 'Paso a paso',
    fieldsTitle: 'Campos y acciones',
    errorsTitle: 'Errores y estados comunes',
    relatedTitle: 'Guías relacionadas',
    fieldDescription: 'Etiqueta o acción visible en esta pantalla.',
    stepOpen: 'Abra **{page}** desde la barra lateral.',
    stepReview: 'Revise los indicadores y la tabla principal.',
    stepAct: 'Use las acciones descritas abajo; los cambios se aplican de inmediato.',
    errorLoad: 'Si los datos no cargan, vuelva a intentarlo. Si persiste, revise el gateway y los proveedores.',
    errorPermission: 'Si una acción está deshabilitada, confirme su rol y el ámbito del tenant.',
    errorEmpty: 'Los estados vacíos son normales antes de la primera solicitud; siga la llamada a la acción.',
    imageAlt: 'Captura de pantalla de {page} en {locale}.'
  },
  fr: {
    whatTitle: 'Ce que fait cet écran',
    whenTitle: 'Quand l’utiliser',
    stepsTitle: 'Étape par étape',
    fieldsTitle: 'Champs et actions',
    errorsTitle: 'Erreurs et états courants',
    relatedTitle: 'Guides associés',
    fieldDescription: 'Libellé ou action visible sur cet écran.',
    stepOpen: 'Ouvrez **{page}** depuis la barre latérale.',
    stepReview: 'Vérifiez les indicateurs et le tableau principal.',
    stepAct: 'Utilisez les actions décrites ci-dessous ; les changements s’appliquent immédiatement.',
    errorLoad: 'Si les données ne se chargent pas, réessayez. Si le problème persiste, vérifiez la passerelle et les fournisseurs.',
    errorPermission: 'Si une action est désactivée, confirmez votre rôle et le périmètre du tenant.',
    errorEmpty: 'Les états vides sont normaux avant la première requête ; suivez l’appel à l’action.',
    imageAlt: 'Capture d’écran de {page} en {locale}.'
  },
  de: {
    whatTitle: 'Was diese Seite tut',
    whenTitle: 'Wann verwenden',
    stepsTitle: 'Schritt für Schritt',
    fieldsTitle: 'Felder und Aktionen',
    errorsTitle: 'Häufige Fehler und Zustände',
    relatedTitle: 'Verwandte Leitfäden',
    fieldDescription: 'Auf dieser Seite sichtbare Bezeichnung oder Aktion.',
    stepOpen: 'Öffnen Sie **{page}** über die Seitenleiste.',
    stepReview: 'Prüfen Sie die Kennzahlen und die Haupttabelle.',
    stepAct: 'Verwenden Sie die unten beschriebenen Aktionen; Änderungen gelten sofort.',
    errorLoad: 'Wenn Daten nicht geladen werden, versuchen Sie es erneut. Prüfen Sie Gateway und Anbieter, wenn der Fehler bleibt.',
    errorPermission: 'Wenn eine Aktion deaktiviert ist, prüfen Sie Rolle und Tenant-Bereich.',
    errorEmpty: 'Leere Zustände sind vor der ersten Anfrage normal; folgen Sie der Handlungsaufforderung.',
    imageAlt: 'Screenshot von {page} auf {locale}.'
  }
};

const GUIDES = [
  {
    slug: 'dashboard', pageKey: 'page.dashboard', related: ['providers', 'observability', 'cost'],
    fields: ['dashboard.kpiActiveKeys', 'dashboard.kpiRequests', 'dashboard.kpiCost', 'dashboard.kpiExpiring', 'dashboard.health'],
    what: {
      en: 'Shows gateway health, traffic, cost and provider status for the last 24 hours.',
      'pt-BR': 'Mostra a saúde do gateway, tráfego, custo e status dos provedores nas últimas 24 horas.',
      'pt-PT': 'Mostra a saúde do gateway, tráfego, custo e estado dos fornecedores nas últimas 24 horas.',
      es: 'Muestra la salud del gateway, el tráfico, el coste y el estado de los proveedores en las últimas 24 horas.',
      fr: 'Affiche la santé de la passerelle, le trafic, les coûts et l’état des fournisseurs sur les dernières 24 heures.',
      de: 'Zeigt Gateway-Zustand, Traffic, Kosten und Anbieterstatus der letzten 24 Stunden.'
    },
    step: {
      en: 'Check provider health and incidents before opening a detailed trace.',
      'pt-BR': 'Verifique a saúde dos provedores e os incidentes antes de abrir um trace detalhado.',
      'pt-PT': 'Verifique a saúde dos fornecedores e os incidentes antes de abrir um trace detalhado.',
      es: 'Revise la salud de los proveedores y los incidentes antes de abrir una traza detallada.',
      fr: 'Vérifiez la santé des fournisseurs et les incidents avant d’ouvrir une trace détaillée.',
      de: 'Prüfen Sie Anbieterzustand und Vorfälle, bevor Sie einen detaillierten Trace öffnen.'
    }
  },
  {
    slug: 'providers', pageKey: 'page.providers', related: ['endpoints', 'dashboard', 'cost'],
    fields: ['common.addProvider', 'providers.grid', 'providers.list', 'common.test', 'common.status'],
    what: {
      en: 'Lists AI providers, accounts, connection status and model availability, with grid/list views and connection tests.',
      'pt-BR': 'Lista provedores de IA, contas, status de conexão e modelos disponíveis, com visões em grade/lista e testes de conexão.',
      'pt-PT': 'Lista fornecedores de IA, contas, estado da ligação e modelos disponíveis, com vistas em grelha/lista e testes de ligação.',
      es: 'Lista proveedores de IA, cuentas, estado de conexión y modelos disponibles, con vistas de cuadrícula/lista y pruebas de conexión.',
      fr: 'Liste les fournisseurs d’IA, les comptes, l’état des connexions et les modèles disponibles, avec vues grille/liste et tests.',
      de: 'Listet KI-Anbieter, Konten, Verbindungsstatus und verfügbare Modelle mit Raster-/Listenansicht und Verbindungstests.'
    },
    step: {
      en: 'Use Grid/List, filter by auth or capability, then test a provider before enabling it.',
      'pt-BR': 'Use Grade/Lista, filtre por autenticação ou capacidade e teste o provedor antes de ativá-lo.',
      'pt-PT': 'Use Grelha/Lista, filtre por autenticação ou capacidade e teste o fornecedor antes de o ativar.',
      es: 'Use Cuadrícula/Lista, filtre por autenticación o capacidad y pruebe el proveedor antes de activarlo.',
      fr: 'Utilisez Grille/Liste, filtrez par authentification ou capacité, puis testez le fournisseur avant activation.',
      de: 'Nutzen Sie Raster/Liste, filtern Sie nach Authentifizierung oder Fähigkeit und testen Sie den Anbieter vor der Aktivierung.'
    }
  },
  {
    slug: 'endpoints', pageKey: 'page.endpoints', related: ['api-manager', 'providers', 'mcp'],
    fields: ['endpoints.baseUrl', 'endpoints.public', 'endpoints.local', 'endpoints.available', 'endpoints.aliases'],
    what: {
      en: 'Shows the final API endpoint, public and local URLs, available routes, CLI aliases and the optional system prompt.',
      'pt-BR': 'Mostra o endpoint final da API, URLs pública e local, rotas disponíveis, aliases para CLIs e o system prompt opcional.',
      'pt-PT': 'Mostra o endpoint final da API, URLs pública e local, rotas disponíveis, aliases para CLIs e o system prompt opcional.',
      es: 'Muestra el endpoint final de la API, URLs pública y local, rutas disponibles, alias para CLI y el system prompt opcional.',
      fr: 'Affiche le point de terminaison final, les URL publiques et locales, les routes, les alias CLI et le system prompt optionnel.',
      de: 'Zeigt den endgültigen API-Endpunkt, öffentliche und lokale URLs, Routen, CLI-Aliase und den optionalen System-Prompt.'
    },
    step: {
      en: 'Copy the public or local URL and use the aliases for CLIs; no tunnels are offered.',
      'pt-BR': 'Copie a URL pública ou local e use os aliases para CLIs; não há opção de túnel.',
      'pt-PT': 'Copie o URL público ou local e use os aliases para CLIs; não existem túneis.',
      es: 'Copie la URL pública o local y use los alias para CLI; no se ofrecen túneles.',
      fr: 'Copiez l’URL publique ou locale et utilisez les alias CLI ; aucun tunnel n’est proposé.',
      de: 'Kopieren Sie die öffentliche oder lokale URL und nutzen Sie CLI-Aliase; Tunnel werden nicht angeboten.'
    }
  },
  {
    slug: 'api-manager', pageKey: 'page.apiManager', related: ['new-api-key', 'endpoints', 'cost'],
    fields: ['common.newKey', 'api.keys', 'api.scopes', 'common.quota', 'api.lastUse', 'common.status'],
    what: {
      en: 'Manages API keys, scopes, quotas, rate limits, expiration and usage by endpoint.',
      'pt-BR': 'Gerencia chaves de API, escopos, cotas, limites de taxa, expiração e uso por endpoint.',
      'pt-PT': 'Gere chaves de API, âmbitos, quotas, limites de taxa, expiração e utilização por endpoint.',
      es: 'Gestiona claves de API, ámbitos, cuotas, límites de tasa, caducidad y uso por endpoint.',
      fr: 'Gère les clés d’API, les portées, les quotas, les limites de débit, l’expiration et l’usage par endpoint.',
      de: 'Verwaltet API-Schlüssel, Bereiche, Kontingente, Ratenlimits, Ablauf und Nutzung pro Endpunkt.'
    },
    step: {
      en: 'Open a key to review scopes and usage, or use + New key to create one.',
      'pt-BR': 'Abra uma chave para revisar escopos e uso, ou use + Nova chave para criar uma.',
      'pt-PT': 'Abra uma chave para rever âmbitos e utilização, ou use + Nova chave para criar uma.',
      es: 'Abra una clave para revisar ámbitos y uso, o use + Nueva clave para crear una.',
      fr: 'Ouvrez une clé pour vérifier les portées et l’usage, ou utilisez + Nouvelle clé pour en créer une.',
      de: 'Öffnen Sie einen Schlüssel für Bereiche und Nutzung oder erstellen Sie mit + Neuer Schlüssel einen neuen.'
    }
  },
  {
    slug: 'mcp', pageKey: 'page.mcp', related: ['api-manager', 'endpoints', 'settings'],
    fields: ['mcp.server', 'mcp.transportMode', 'mcp.activity', 'mcp.toolsAndScopes', 'common.restart'],
    what: {
      en: 'Controls the MCP server: global on/off, transport, status, activity and exposed tools.',
      'pt-BR': 'Controla o servidor MCP: ativação global, transporte, status, atividade e ferramentas expostas.',
      'pt-PT': 'Controla o servidor MCP: ativação global, transporte, estado, atividade e ferramentas expostas.',
      es: 'Controla el servidor MCP: activación global, transporte, estado, actividad y herramientas expuestas.',
      fr: 'Contrôle le serveur MCP : activation globale, transport, état, activité et outils exposés.',
      de: 'Steuert den MCP-Server: globales Ein/Aus, Transport, Status, Aktivität und bereitgestellte Werkzeuge.'
    },
    step: {
      en: 'Enable the server, choose a transport and copy the command or URL.',
      'pt-BR': 'Ative o servidor, escolha um transporte e copie o comando ou a URL.',
      'pt-PT': 'Ative o servidor, escolha um transporte e copie o comando ou o URL.',
      es: 'Active el servidor, elija un transporte y copie el comando o la URL.',
      fr: 'Activez le serveur, choisissez un transport et copiez la commande ou l’URL.',
      de: 'Aktivieren Sie den Server, wählen Sie einen Transport und kopieren Sie Befehl oder URL.'
    }
  },
  {
    slug: 'cache', pageKey: 'page.cache', related: ['cost', 'observability', 'dlp'],
    fields: ['cache.hitRate', 'cache.savings24h', 'cache.layers', 'cache.recentEvents', 'common.invalidate'],
    what: {
      en: 'Shows exact and semantic cache layers, hit rate, savings and invalidation actions.',
      'pt-BR': 'Mostra as camadas de cache exato e semântico, hit rate, economia e ações de invalidação.',
      'pt-PT': 'Mostra as camadas de cache exato e semântico, hit rate, poupança e ações de invalidação.',
      es: 'Muestra las capas de caché exacta y semántica, hit rate, ahorro y acciones de invalidación.',
      fr: 'Affiche les couches de cache exact et sémantique, le hit rate, les économies et l’invalidation.',
      de: 'Zeigt exakte und semantische Cache-Schichten, Hit-Rate, Einsparungen und Invalidierungsaktionen.'
    },
    step: {
      en: 'Review hit rate and savings, then invalidate by model, provider or tenant if needed.',
      'pt-BR': 'Revise o hit rate e a economia e invalide por modelo, provedor ou tenant se necessário.',
      'pt-PT': 'Reveja o hit rate e a poupança e invalide por modelo, fornecedor ou tenant se necessário.',
      es: 'Revise el hit rate y el ahorro e invalide por modelo, proveedor o tenant si es necesario.',
      fr: 'Vérifiez le hit rate et les économies, puis invalidez par modèle, fournisseur ou tenant si nécessaire.',
      de: 'Prüfen Sie Hit-Rate und Einsparungen und invalidieren Sie bei Bedarf nach Modell, Anbieter oder Tenant.'
    }
  },
  {
    slug: 'dlp', pageKey: 'page.dlp', related: ['observability', 'api-manager', 'cache'],
    fields: ['dlp.rules', 'common.action', 'common.severity', 'common.status', 'dlp.runTest'],
    what: {
      en: 'Defines DLP rules and policies, tests redaction and audits events without exposing PII.',
      'pt-BR': 'Define regras e políticas de DLP, testa a redação e audita eventos sem expor PII.',
      'pt-PT': 'Define regras e políticas de DLP, testa a redação e audita eventos sem expor PII.',
      es: 'Define reglas y políticas de DLP, prueba el enmascaramiento y audita eventos sin exponer PII.',
      fr: 'Définit les règles et politiques DLP, teste le masquage et audite les événements sans exposer de données personnelles.',
      de: 'Definiert DLP-Regeln und -Richtlinien, testet Maskierung und auditiert Ereignisse ohne Offenlegung personenbezogener Daten.'
    },
    step: {
      en: 'Run a rule test with sample text and confirm the action before enabling it.',
      'pt-BR': 'Execute um teste de regra com texto de exemplo e confirme a ação antes de ativá-la.',
      'pt-PT': 'Execute um teste de regra com texto de exemplo e confirme a ação antes de a ativar.',
      es: 'Ejecute una prueba de regla con texto de ejemplo y confirme la acción antes de activarla.',
      fr: 'Lancez un test de règle avec un texte d’exemple et confirmez l’action avant activation.',
      de: 'Führen Sie einen Regeltest mit Beispieltext aus und bestätigen Sie die Aktion vor der Aktivierung.'
    }
  },
  {
    slug: 'observability', pageKey: 'page.observability', related: ['dashboard', 'dlp', 'cost'],
    fields: ['observability.availability', 'observability.slo', 'observability.alerts', 'observability.traffic', 'common.openBackend'],
    what: {
      en: 'Shows SLOs, availability, alerts, traffic and latency, with deep links to the managed backend.',
      'pt-BR': 'Mostra SLOs, disponibilidade, alertas, tráfego e latência, com deep links para o backend gerenciado.',
      'pt-PT': 'Mostra SLOs, disponibilidade, alertas, tráfego e latência, com deep links para o backend gerido.',
      es: 'Muestra SLOs, disponibilidad, alertas, tráfico y latencia, con deep links al backend gestionado.',
      fr: 'Affiche les SLO, la disponibilité, les alertes, le trafic et la latence, avec des liens vers le backend managé.',
      de: 'Zeigt SLOs, Verfügbarkeit, Warnungen, Traffic und Latenz mit Deep Links zum verwalteten Backend.'
    },
    step: {
      en: 'Use the deep links to open traces and SLO details in the managed backend.',
      'pt-BR': 'Use os deep links para abrir traces e detalhes de SLO no backend gerenciado.',
      'pt-PT': 'Use os deep links para abrir traces e detalhes de SLO no backend gerido.',
      es: 'Use los deep links para abrir trazas y detalles de SLO en el backend gestionado.',
      fr: 'Utilisez les liens pour ouvrir les traces et le détail des SLO dans le backend managé.',
      de: 'Nutzen Sie die Deep Links, um Traces und SLO-Details im verwalteten Backend zu öffnen.'
    }
  },
  {
    slug: 'cost', pageKey: 'page.cost', related: ['dashboard', 'api-manager', 'cache'],
    fields: ['cost.today', 'cost.month', 'cost.projection', 'cost.budget', 'cost.quotas'],
    what: {
      en: 'Shows daily and monthly cost, projection, budget and quotas by tenant, key and provider.',
      'pt-BR': 'Mostra custo diário e mensal, projeção, orçamento e cotas por tenant, chave e provedor.',
      'pt-PT': 'Mostra custo diário e mensal, projeção, orçamento e quotas por tenant, chave e fornecedor.',
      es: 'Muestra coste diario y mensual, proyección, presupuesto y cuotas por tenant, clave y proveedor.',
      fr: 'Affiche les coûts journaliers et mensuels, la projection, le budget et les quotas par tenant, clé et fournisseur.',
      de: 'Zeigt Tages- und Monatskosten, Prognose, Budget und Kontingente pro Tenant, Schlüssel und Anbieter.'
    },
    step: {
      en: 'Set quotas and budgets, then export the chargeback report.',
      'pt-BR': 'Defina cotas e orçamentos e exporte o relatório de chargeback.',
      'pt-PT': 'Defina quotas e orçamentos e exporte o relatório de chargeback.',
      es: 'Defina cuotas y presupuestos y exporte el informe de chargeback.',
      fr: 'Définissez quotas et budgets, puis exportez le rapport de refacturation.',
      de: 'Legen Sie Kontingente und Budgets fest und exportieren Sie den Verrechnungsbericht.'
    }
  },
  {
    slug: 'settings', pageKey: 'page.settings', related: ['onboarding', 'api-manager', 'mcp'],
    fields: ['settings.language', 'settings.preference', 'settings.storageKey', 'settings.fallbackNote', 'settings.openGuide'],
    what: {
      en: 'Lets you change the interface language and review where the preference is stored.',
      'pt-BR': 'Permite alterar o idioma da interface e revisar onde a preferência é armazenada.',
      'pt-PT': 'Permite alterar o idioma da interface e rever onde a preferência é guardada.',
      es: 'Permite cambiar el idioma de la interfaz y revisar dónde se guarda la preferencia.',
      fr: 'Permet de changer la langue de l’interface et de vérifier où la préférence est stockée.',
      de: 'Ermöglicht das Ändern der Oberflächensprache und zeigt, wo die Präferenz gespeichert wird.'
    },
    step: {
      en: 'Pick a language; the interface updates immediately and the choice is saved.',
      'pt-BR': 'Escolha um idioma; a interface atualiza imediatamente e a escolha é salva.',
      'pt-PT': 'Escolha um idioma; a interface atualiza imediatamente e a escolha é guardada.',
      es: 'Elija un idioma; la interfaz se actualiza de inmediato y la elección se guarda.',
      fr: 'Choisissez une langue ; l’interface se met à jour immédiatement et le choix est enregistré.',
      de: 'Wählen Sie eine Sprache; die Oberfläche aktualisiert sich sofort und die Auswahl wird gespeichert.'
    }
  },
  {
    slug: 'new-api-key', pageKey: 'page.newApiKey', related: ['api-manager', 'endpoints', 'cost'],
    fields: ['newKey.name', 'newKey.costCenter', 'newKey.expiration', 'newKey.quota', 'newKey.allowedEndpoints', 'newKey.allowedModels'],
    what: {
      en: 'Creates an API key with scopes, model restrictions, quotas, expiration and IP/session limits.',
      'pt-BR': 'Cria uma chave de API com escopos, restrições de modelo, cotas, expiração e limites de IP/sessão.',
      'pt-PT': 'Cria uma chave de API com âmbitos, restrições de modelo, quotas, expiração e limites de IP/sessão.',
      es: 'Crea una clave de API con ámbitos, restricciones de modelo, cuotas, caducidad y límites de IP/sesión.',
      fr: 'Crée une clé d’API avec portées, restrictions de modèles, quotas, expiration et limites IP/session.',
      de: 'Erstellt einen API-Schlüssel mit Bereichen, Modellbeschränkungen, Kontingenten, Ablauf und IP-/Sitzungslimits.'
    },
    step: {
      en: 'Fill in the fields, select models, create the key and copy it once.',
      'pt-BR': 'Preencha os campos, selecione modelos, crie a chave e copie-a uma única vez.',
      'pt-PT': 'Preencha os campos, selecione modelos, crie a chave e copie-a uma única vez.',
      es: 'Complete los campos, seleccione modelos, cree la clave y cópiela una sola vez.',
      fr: 'Remplissez les champs, sélectionnez les modèles, créez la clé et copiez-la une seule fois.',
      de: 'Füllen Sie die Felder aus, wählen Sie Modelle, erstellen Sie den Schlüssel und kopieren Sie ihn einmal.'
    }
  },
  {
    slug: 'onboarding', pageKey: 'page.onboarding', related: ['settings', 'dashboard', 'api-manager'],
    fields: ['onboarding.languageLegend', 'onboarding.detected', 'onboarding.confirm', 'onboarding.help'],
    what: {
      en: 'Appears on first access to choose the interface language, pre-selected from the browser.',
      'pt-BR': 'Aparece no primeiro acesso para escolher o idioma da interface, pré-selecionado pelo navegador.',
      'pt-PT': 'Aparece no primeiro acesso para escolher o idioma da interface, pré-selecionado pelo navegador.',
      es: 'Aparece en el primer acceso para elegir el idioma de la interfaz, preseleccionado por el navegador.',
      fr: 'Apparaît au premier accès pour choisir la langue de l’interface, présélectionnée par le navigateur.',
      de: 'Erscheint beim ersten Zugriff zur Auswahl der Oberflächensprache, vorausgewählt aus dem Browser.'
    },
    step: {
      en: 'Choose the suggested language or another one, then select Continue.',
      'pt-BR': 'Escolha o idioma sugerido ou outro e selecione Continuar.',
      'pt-PT': 'Escolha o idioma sugerido ou outro e selecione Continuar.',
      es: 'Elija el idioma sugerido u otro y seleccione Continuar.',
      fr: 'Choisissez la langue suggérée ou une autre, puis sélectionnez Continuer.',
      de: 'Wählen Sie die vorgeschlagene oder eine andere Sprache und klicken Sie auf Weiter.'
    }
  }
];

module.exports = { SHARED, GUIDES };
