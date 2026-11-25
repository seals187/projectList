// Checks the hand-written frontend types against the HTTP contract generated
// from Go (api.gen.ts, `go run ./cmd/gentypes`). The type checks run in
// svelte-check: a field the web app reads but the API no longer sends, or an
// enum value that differs, is a compile error here.
import { describe, expect, it } from 'vitest';
import { ERROR_CODES } from './api.gen';
import type * as Api from './api.gen';
import type * as T from './types';
import { tMaybe } from './i18n.svelte';

/** Compiles only if A and B are the same type. */
type Equal<A, B> = (<X>() => X extends A ? 1 : 2) extends <X>() => X extends B ? 1 : 2 ? true : false;
/** Fields of the frontend type H that the API type G does not have. */
type Unknown<H, G> = Exclude<keyof H, keyof G>;
const assert = <V extends true>() => {};
const none = <K extends never>() => {};

// enums
assert<Equal<T.Role, Api.Role>>();
assert<Equal<T.ProjectStatus, Api.ProjectStatus>>();
assert<Equal<T.Kind, Api.Kind>>();
assert<Equal<T.NoteColor, Api.NoteColor>>();
assert<Equal<T.FavoriteKind, Api.FavoriteKind>>();
assert<Equal<T.TemplateKind, Api.TemplateKind>>();

// response shapes: every field the web app expects is sent by the API
none<Unknown<T.AdminUser, Api.ApiUser>>();
none<Unknown<T.Customer, Api.ApiCustomer>>();
none<Unknown<T.Project, Api.ApiProject>>();
none<Unknown<T.Tag, Api.ApiTag>>();
none<Unknown<T.Item, Api.ApiItem>>();
none<Unknown<T.TemplateItem, Api.ApiTemplateItem>>();
none<Unknown<T.Checklist, Api.ApiChecklist>>();
none<Unknown<T.ActivityEntry, Api.ApiActivityEntry>>();
none<Unknown<T.Table, Api.ApiTable>>();

/** Codes for programming errors of the web app itself; never shown with their own text. */
const UNTRANSLATED = new Set(['invalid customer_id', 'invalid scope']);

describe('API contract', () => {
  it('every error code the API can return has a translation', () => {
    const missing = ERROR_CODES.filter((c) => !UNTRANSLATED.has(c) && tMaybe(`errors.${c}`) === null);
    expect(missing).toEqual([]);
  });
});
