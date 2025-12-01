use axum::{
    http::{header, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
};
use jsonwebtoken::{decode, DecodingKey, Validation};

use crate::config::JWT_SECRET;
use crate::types::Claims;

pub async fn auth_middleware(
    headers: axum::http::HeaderMap,
    request: axum::extract::Request,
    next: Next,
) -> Response {
    if let Some(auth) = headers
        .get(header::AUTHORIZATION)
        .and_then(|h| h.to_str().ok())
    {
        if let Some(token) = auth.strip_prefix("Bearer ") {
            if decode::<Claims>(
                token,
                &DecodingKey::from_secret(JWT_SECRET.as_bytes()),
                &Validation::default(),
            )
            .is_ok()
            {
                return next.run(request).await;
            }
        }
    }
    StatusCode::UNAUTHORIZED.into_response()
}

