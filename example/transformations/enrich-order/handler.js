addHandler("transform", (request, context) => {
  const body = request.body;

  // Use environment-specific variables from the manifest's env config
  body.enrichment_api = context.env.API_BASE_URL;
  body.processed_at = new Date().toISOString();
  body.order_total_cents = Math.round(body.order_total * 100);

  request.body = body;
  return request;
});
