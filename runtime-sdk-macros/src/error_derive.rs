use darling::{util::Flag, FromDeriveInput, FromVariant};
use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{DeriveInput, Ident};

use crate::generators::{self as gen, CodedVariant};

#[derive(FromDeriveInput)]
#[darling(supports(enum_any), attributes(sdk_error))]
struct Error {
    ident: Ident,

    data: darling::ast::Data<ErrorVariant, darling::util::Ignored>,

    /// The path to the module type.
    #[darling(default)]
    module: Option<syn::Path>,

    /// The path to a const set to the module name.
    /// This is intended for use only by core modules.
    #[darling(default)]
    module_name_path: Option<syn::Path>,

    /// Whether to sequentially autonumber the error codes.
    /// This option exists as a convenience for runtimes that
    /// only append errors or release only breaking changes.
    #[darling(default, rename = "autonumber")]
    autonumber: Flag,
}

#[derive(FromVariant)]
#[darling(attributes(sdk_error))]
struct ErrorVariant {
    ident: Ident,

    /// The explicit ID of the error code. Overrides any autonumber set on the error enum.
    #[darling(default, rename = "code")]
    code: Option<u32>,
}

impl CodedVariant for ErrorVariant {
    const FIELD_NAME: &'static str = "code";

    fn ident(&self) -> &Ident {
        &self.ident
    }

    fn code(&self) -> Option<u32> {
        self.code
    }
}

pub fn derive_error(input: DeriveInput) -> TokenStream {
    let error = match Error::from_derive_input(&input) {
        Ok(error) => error,
        Err(e) => return e.write_errors(),
    };

    let error_ty_ident = &error.ident;

    let code_converter = gen::enum_code_converter(
        &format_ident!("self"),
        &error.data.as_ref().take_enum().unwrap(),
        error.autonumber.is_some(),
    );

    let sdk_crate = gen::sdk_crate_path();

    let module_name = match (&error.module, &error.module_name_path) {
        (Some(module_path), None) => quote!(<#module_path as #sdk_crate::module::Module>::NAME),
        (None, Some(module_name)) => quote!(#module_name),
        (Some(_), Some(_)) => quote!(compile_error!(
            "either `module` and `module_name` must be set"
        )),
        (None, None) => quote!(compile_error!(
            r#"missing `#[sdk_error(module = "path::to::Module")]` attribute"#
        )),
    };

    gen::wrap_in_const(quote! {
        impl #sdk_crate::error::Error for #error_ty_ident {
            fn module(&self) -> &str {
                #module_name
            }

            fn code(&self) -> u32 {
                #code_converter
            }
        }
    })
}

#[cfg(test)]
mod tests {
    #[test]
    fn generate_error_impl() {
        let expected: syn::Stmt = syn::parse_quote!(
            const _: () = {
                impl ::oasis_runtime_sdk::error::Error for Error {
                    fn module(&self) -> &str {
                        <module::TheModule as ::oasis_runtime_sdk::module::Module>::NAME
                    }
                    fn code(&self) -> u32 {
                        match self {
                            Self::Error0 { .. } => 0u32,
                            Self::Error2 { .. } => 2u32,
                            Self::Error1 { .. } => 1u32,
                            Self::Error3 { .. } => 3u32,
                        }
                    }
                }
            };
        );

        let input: syn::DeriveInput = syn::parse_quote!(
            #[derive(Error)]
            #[sdk_error(autonumber, module = "module::TheModule")]
            pub enum Error {
                Error0,
                #[sdk_error(code = 2)]
                Error2 {
                    payload: Vec<u8>,
                },
                Error1(String),
                Error3,
            }
        );
        let error_derivation = super::derive_error(input);
        let actual: syn::Stmt = syn::parse2(error_derivation).unwrap();

        crate::assert_empty_diff!(actual, expected);
    }
}
