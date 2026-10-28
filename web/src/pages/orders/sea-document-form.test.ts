import { describe, expect, it } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  buildSeaDocumentUpdate,
  mergeSeaDocumentBaseline,
  seaDocumentFormValues,
} from './sea-document-form';

const documents: API.SeaOrderDocuments = {
  linkVersion: '4',
  documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
  masterBill: { id: 'm', version: '6', content: { shipperText: '原主单' } },
  houseBill: {
    id: 'h',
    houseNo: 'HBL',
    issuerSource: 1,
    version: '2',
    status: 1,
    content: { shipperText: '原分单' },
  },
};
describe('海运统一保存载荷', () => {
  it('原始单证不提交，元数据不影响变更判定', () => {
    const values = seaDocumentFormValues(documents);
    expect(buildSeaDocumentUpdate(values, documents)).toBeUndefined();
    if (values.seaHouseBill) {
      delete values.seaHouseBill.expectedVersion;
      delete values.seaHouseBill.id;
      delete values.seaHouseBill.status;
    }
    expect(buildSeaDocumentUpdate(values, documents)).toBeUndefined();
  });
  it('两份正文修改统一携带各自版本及Link，显式清空得到保留', () => {
    const values = seaDocumentFormValues(documents);
    values.seaMasterBillContent = {
      ...values.seaMasterBillContent,
      shipperText: '',
    };
    if (values.seaHouseBill) values.seaHouseBill.houseNo = 'NEW-HBL';
    const update = buildSeaDocumentUpdate(values, documents);
    expect(update).toMatchObject({
      expectedMblVersion: '6',
      masterBillContent: { shipperText: '' },
      expectedLinkVersion: '4',
      houseBill: { id: 'h', houseNo: 'NEW-HBL', expectedVersion: '2' },
    });
    expect(update?.houseBill).not.toHaveProperty('status');
    expect(update).not.toHaveProperty('documentStructure');
  });
  it('只改HBL不提交共享主单，正文前后空白修改不会被吞掉', () => {
    const values = seaDocumentFormValues(documents);
    if (values.seaHouseBill?.content)
      values.seaHouseBill.content.shipperText = ' 原分单 ';
    expect(
      buildSeaDocumentUpdate(values, documents)?.masterBillContent,
    ).toBeUndefined();
    expect(
      buildSeaDocumentUpdate(values, documents)?.houseBill?.content
        ?.shipperText,
    ).toBe(' 原分单 ');
  });
  it('DIRECT不构造隐藏分单', () => {
    const direct = { ...documents, houseBill: undefined, documentStructure: 1 };
    const values = seaDocumentFormValues(direct);
    values.seaMasterBillContent = {
      ...values.seaMasterBillContent,
      shipperText: '新主单',
    };
    expect(buildSeaDocumentUpdate(values, direct)?.houseBill).toBeUndefined();
  });
  it('恢复草稿保持原MBL/HBL/Link版本，不能采用别人更新后的最新版覆盖', () => {
    const draft = seaDocumentFormValues(documents);
    draft.seaMasterBillContent = {
      ...draft.seaMasterBillContent,
      shipperText: '草稿主单',
    };
    if (draft.seaHouseBill) draft.seaHouseBill.houseNo = '草稿分单';
    const latest: API.SeaOrderDocuments = {
      ...documents,
      linkVersion: '8',
      masterBill: {
        ...documents.masterBill,
        version: '10',
        content: { shipperText: '他人更新主单' },
      },
      houseBill: {
        ...documents.houseBill,
        version: '11',
        houseNo: '他人更新分单',
      },
    };
    expect(buildSeaDocumentUpdate(draft, latest)).toMatchObject({
      expectedMblVersion: '6',
      expectedLinkVersion: '4',
      houseBill: { id: 'h', expectedVersion: '2', houseNo: '草稿分单' },
    });
  });

  it.each(['master', 'house', 'link'] as const)(
    '修改单证但缺少%s原版本时明确阻断',
    (part) => {
      const draft = seaDocumentFormValues(documents);
      draft.seaMasterBillContent = {
        ...draft.seaMasterBillContent,
        shipperText: '修改主单',
      };
      if (draft.seaHouseBill) draft.seaHouseBill.houseNo = '修改分单';
      if (part === 'master') delete draft.seaMasterBillContentVersion;
      if (part === 'house' && draft.seaHouseBill)
        delete draft.seaHouseBill.expectedVersion;
      if (part === 'link') delete draft.seaDocumentLinkVersion;
      expect(() => buildSeaDocumentUpdate(draft, documents)).toThrow();
    },
  );

  it('模式变更后的新HBL不能成为旧草稿的写入目标', () => {
    const draft = seaDocumentFormValues(documents);
    if (draft.seaHouseBill) draft.seaHouseBill.houseNo = '旧目标草稿';
    const newTarget = {
      ...documents,
      houseBill: { ...documents.houseBill, id: 'new-house', version: '1' },
    };
    expect(() => buildSeaDocumentUpdate(draft, newTarget)).toThrow();
  });

  it.each(['master-first', 'house-first'] as const)(
    '并发专用刷新%s合并各自范围，不回退另一份版本',
    (order) => {
      const masterSnapshot = {
        ...documents,
        masterBill: {
          ...documents.masterBill,
          version: '7',
          content: { shipperText: '更新主单' },
        },
      };
      const houseSnapshot = {
        ...documents,
        linkVersion: '5',
        houseBill: {
          ...documents.houseBill,
          version: '3',
          houseNo: '更新分单',
        },
      };
      const first =
        order === 'master-first'
          ? mergeSeaDocumentBaseline(documents, masterSnapshot, 'master')
          : mergeSeaDocumentBaseline(documents, houseSnapshot, 'house');
      const result =
        order === 'master-first'
          ? mergeSeaDocumentBaseline(first, houseSnapshot, 'house')
          : mergeSeaDocumentBaseline(first, masterSnapshot, 'master');
      expect(result).toMatchObject({
        masterBill: { version: '7' },
        houseBill: { version: '3' },
        linkVersion: '5',
      });
    },
  );
});
