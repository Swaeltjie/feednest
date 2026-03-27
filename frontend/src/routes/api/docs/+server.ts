import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>FeedNest API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui.css" integrity="sha256-VAnDgCFieKLfi0VXWwjhEVMtSKdIWLBrKJoe7MoPPiA=" crossorigin="anonymous">
  <style>
    body { margin: 0; }
    .swagger-ui .topbar { display: none; }
    .swagger-ui { max-width: 1200px; margin: 0 auto; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui-bundle.js" integrity="sha256-89+x4m2xIxMHfJU4GHnevKwNf5mYuJSVENLIkqFpJGE=" crossorigin="anonymous"></script>
  <script>
    SwaggerUIBundle({
      url: '/api/docs/openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout',
    });
  </script>
</body>
</html>`;
	return new Response(html, {
		headers: { 'Content-Type': 'text/html; charset=utf-8' },
	});
};
