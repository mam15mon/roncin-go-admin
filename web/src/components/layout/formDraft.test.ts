import dayjs, { type Dayjs } from 'dayjs';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  captureFormDraftSession,
  clearFormDraft,
  clearTabDrafts,
  getFormDraft,
  getFormDraftKey,
  getFormDraftScope,
  hasFormDraft,
  hasTabDraft,
  prepareFormDraftClear,
  reviveFormDraft,
  saveFormDraft,
} from './formDraft';

describe('formDraft', () => {
  const draftScope = getFormDraftScope('user-1', 'org-1');

  beforeEach(() => {
    sessionStorage.clear();
  });

  describe('getFormDraftKey', () => {
    it('constructs predictable draft keys', () => {
      expect(
        getFormDraftKey(
          '/orders/sea-export',
          '/orders/sea-export/new',
          draftScope,
        ),
      ).toBe(
        'roncin:tab-draft:user-1:org-1:%2Forders%2Fsea-export:/orders/sea-export/new',
      );
      expect(
        getFormDraftKey(
          '/orders/sea-export',
          '/orders/sea-export/123',
          draftScope,
        ),
      ).toBe(
        'roncin:tab-draft:user-1:org-1:%2Forders%2Fsea-export:/orders/sea-export/123',
      );
      expect(getFormDraftKey('/orders/sea-export', '/orders/new')).toBe('');
      expect(getFormDraftKey('/orders/sea-export', undefined, draftScope)).toBe(
        '',
      );
    });
  });

  describe('reviveFormDraft', () => {
    it('revives known date fields into dayjs instances', () => {
      const nowStr = '2026-09-08T15:30:00.000Z';
      const input = {
        customerReferenceNo: 'REF-888',
        orderDate: nowStr,
        etd: '2026-09-10',
        declarationCutoffAt: '2026-09-09T08:00:00.000Z',
        receivedAt: '2026-09-08T09:00:00.000Z',
        nested: {
          siCutoff: '2026-09-09T18:00:00.000Z',
          cargoReadyAt: '2026-09-09T12:00:00.000Z',
        },
        confirmations: [{ confirmedAt: '2026-09-09T10:00:00.000Z' }],
        referenceText: '2026-09-08T15:30:00.000Z',
        regularText: '2026-09-08 is a good day',
      };

      const revived = reviveFormDraft(input);
      expect(dayjs.isDayjs(revived.orderDate)).toBe(true);
      expect(dayjs.isDayjs(revived.etd)).toBe(true);
      expect(dayjs.isDayjs(revived.declarationCutoffAt)).toBe(true);
      expect(dayjs.isDayjs(revived.receivedAt)).toBe(true);
      expect(dayjs.isDayjs(revived.nested.siCutoff)).toBe(true);
      expect(dayjs.isDayjs(revived.nested.cargoReadyAt)).toBe(true);
      expect(dayjs.isDayjs(revived.confirmations[0].confirmedAt)).toBe(true);
      expect(revived.customerReferenceNo).toBe('REF-888');
      expect(revived.referenceText).toBe('2026-09-08T15:30:00.000Z');
      expect(revived.regularText).toBe('2026-09-08 is a good day');
    });
  });

  describe('saveFormDraft and getFormDraft', () => {
    it('saves and reads draft with dates revived', () => {
      const draftKey = getFormDraftKey(
        '/orders/sea-export',
        '/orders/sea-export/new',
        draftScope,
      );
      const data = {
        customerReferenceNo: 'TEST-REF',
        orderDate: dayjs('2026-09-08'),
        remarks: 'Handle with care',
      };

      saveFormDraft(draftKey, data);
      expect(hasFormDraft(draftKey)).toBe(true);

      const loaded = getFormDraft<{
        customerReferenceNo: string;
        orderDate: Dayjs;
        remarks: string;
      }>(draftKey);
      expect(loaded).not.toBeNull();
      expect(loaded?.customerReferenceNo).toBe('TEST-REF');
      expect(loaded?.remarks).toBe('Handle with care');
      expect(dayjs.isDayjs(loaded?.orderDate)).toBe(true);
      expect(loaded?.orderDate.format('YYYY-MM-DD')).toBe('2026-09-08');
    });

    it('returns null for non-existent draft', () => {
      expect(getFormDraft('non-existent')).toBeNull();
      expect(hasFormDraft('non-existent')).toBe(false);
    });
  });

  describe('hasTabDraft and clearTabDrafts', () => {
    it('忽略旧格式草稿，支持当前页签的多个编辑地址', () => {
      const tabKey = '/orders/sea-export';
      const oldKey =
        'roncin:form-draft:user-1:org-1:/orders/sea-export:/orders/sea-export/new';
      sessionStorage.setItem(oldKey, '{"note":"旧格式"}');
      expect(hasTabDraft(tabKey, draftScope)).toBe(false);

      const key = getFormDraftKey(
        tabKey,
        '/orders/sea-export/order-1',
        draftScope,
      );
      saveFormDraft(key, { note: '当前详情草稿' });
      expect(hasTabDraft(tabKey, draftScope)).toBe(true);
    });

    it('确认关闭使旧实例的写入与清理失效，重开同一地址不受影响', () => {
      const tabKey = '/partners/customers';
      const key = getFormDraftKey(
        tabKey,
        '/partners/customers/new',
        draftScope,
      );
      const oldSession = captureFormDraftSession(key);
      saveFormDraft(key, { note: '待放弃' }, oldSession);

      clearTabDrafts(tabKey, draftScope);
      expect(saveFormDraft(key, { note: '迟到快照' }, oldSession)).toBe(false);
      expect(hasFormDraft(key)).toBe(false);

      const newSession = captureFormDraftSession(key);
      saveFormDraft(key, { note: '重新编辑' }, newSession);
      clearFormDraft(key, oldSession);
      expect(getFormDraft(key)).toEqual({ note: '重新编辑' });
    });

    it('后台保存只清理原快照，保留后续编辑或重新打开的页签', () => {
      const tabKey = '/partners/customers';
      const key = getFormDraftKey(
        tabKey,
        '/partners/customers/new',
        draftScope,
      );
      const session = captureFormDraftSession(key);
      saveFormDraft(key, { note: '待提交' }, session);
      const clearSubmitted = prepareFormDraftClear(key, session);
      saveFormDraft(key, { note: '新编辑' }, session);
      clearSubmitted();
      expect(getFormDraft(key)).toEqual({ note: '新编辑' });

      const clearLater = prepareFormDraftClear(key);
      clearTabDrafts(tabKey, draftScope);
      const reopened = captureFormDraftSession(key);
      saveFormDraft(key, { note: '新编辑' }, reopened);
      clearLater();
      expect(getFormDraft(key)).toEqual({ note: '新编辑' });

      prepareFormDraftClear(key, reopened)();
      expect(hasFormDraft(key)).toBe(false);
    });

    it('detects and clears drafts belonging to a specific tabKey', () => {
      const tabKey = '/orders/sea-export';
      const key1 = getFormDraftKey(
        tabKey,
        '/orders/sea-export/new',
        draftScope,
      );
      const key2 = getFormDraftKey(
        tabKey,
        '/orders/sea-export/123',
        draftScope,
      );
      const otherKey = getFormDraftKey(
        '/orders/air-export/new',
        '/orders/air-export/new',
        draftScope,
      );

      saveFormDraft(key1, { field1: 'val1' });
      saveFormDraft(key2, { field2: 'val2' });
      saveFormDraft(otherKey, { field3: 'val3' });

      expect(hasTabDraft(tabKey, draftScope)).toBe(true);
      expect(hasTabDraft('/orders/air-export/new', draftScope)).toBe(true);

      clearTabDrafts(tabKey, draftScope);

      expect(hasTabDraft(tabKey, draftScope)).toBe(false);
      expect(hasFormDraft(key1)).toBe(false);
      expect(hasFormDraft(key2)).toBe(false);
      // Other tab's draft is untouched
      expect(hasTabDraft('/orders/air-export/new', draftScope)).toBe(true);
      expect(hasFormDraft(otherKey)).toBe(true);
    });

    it('相同页签的草稿按用户与组织隔离', () => {
      const tabKey = '/orders/sea-export';
      const otherScope = getFormDraftScope('user-1', 'org-2');
      const currentKey = getFormDraftKey(
        tabKey,
        '/orders/sea-export/new',
        draftScope,
      );
      const otherKey = getFormDraftKey(
        tabKey,
        '/orders/sea-export/new',
        otherScope,
      );
      saveFormDraft(currentKey, { customerReferenceNo: 'ORG-1' });
      saveFormDraft(otherKey, { customerReferenceNo: 'ORG-2' });

      clearTabDrafts(tabKey, draftScope);

      expect(hasTabDraft(tabKey, draftScope)).toBe(false);
      expect(hasTabDraft(tabKey, otherScope)).toBe(true);
      expect(getFormDraft(otherKey)).toEqual({
        customerReferenceNo: 'ORG-2',
      });
    });
  });

  describe('clearFormDraft', () => {
    it('clears specific draft by key', () => {
      const draftKey = 'roncin:form-draft:test';
      saveFormDraft(draftKey, { foo: 'bar' });
      expect(hasFormDraft(draftKey)).toBe(true);

      clearFormDraft(draftKey);
      expect(hasFormDraft(draftKey)).toBe(false);
    });
  });
});
