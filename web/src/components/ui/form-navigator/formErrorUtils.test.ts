import {
  collectFormSectionErrors,
  findFieldDomElement,
  findParentSectionKey,
  getFormItemLabel,
  scrollToFirstFormError,
  scrollToFirstTableError,
} from './formErrorUtils';

describe('formErrorUtils', () => {
  it('可以正确从 DOM 中查找父级 SectionCard 的 sectionKey 或 id', () => {
    const wrapper = document.createElement('div');
    wrapper.innerHTML = `
      <div class="roncin-section-card" data-section-key="basicInfo" id="section-basicInfo">
        <div class="ant-form-item ant-form-item-has-error">
          <div class="ant-form-item-label"><label>订单编号</label></div>
          <input id="form_orderNo" name="orderNo" />
        </div>
      </div>
    `;
    document.body.appendChild(wrapper);

    const input = document.getElementById('form_orderNo') as HTMLElement;
    expect(findParentSectionKey(input)).toBe('basicInfo');

    const formItem = input.closest('.ant-form-item') as HTMLElement;
    expect(getFormItemLabel(formItem)).toBe('订单编号');

    document.body.removeChild(wrapper);
  });

  it('collectFormSectionErrors 能够准确统计各分节下的错误总数', () => {
    const container = document.createElement('div');
    container.innerHTML = `
      <div data-section-key="sectionA">
        <div class="ant-form-item ant-form-item-has-error"></div>
        <div class="ant-form-item ant-form-item-has-error"></div>
      </div>
      <div data-section-key="sectionB">
        <div class="ant-form-item ant-form-item-has-error"></div>
      </div>
    `;

    const stats = collectFormSectionErrors(container);
    expect(stats).toEqual({
      sectionA: 2,
      sectionB: 1,
    });
  });

  it('scrollToFirstFormError 能够提取首个错误，触发自动展开并平滑滚动', () => {
    const container = document.createElement('div');
    container.innerHTML = `
      <div class="roncin-section-card" data-section-key="transport">
        <div class="ant-form-item ant-form-item-has-error">
          <div class="ant-form-item-label"><label>船公司</label></div>
          <input id="orderForm_shippingLineId" name="shippingLineId" />
        </div>
      </div>
    `;
    document.body.appendChild(container);

    const onExpandSection = vi.fn();
    const notify = vi.fn();

    // 模拟 window.scrollTo
    const originalScrollTo = window.scrollTo;
    window.scrollTo = vi.fn();

    const result = scrollToFirstFormError({
      errorFields: [{ name: ['shippingLineId'], errors: ['请选择船公司'] }],
      container,
      onExpandSection,
      notify,
    });

    expect(result.success).toBe(true);
    expect(result.totalErrors).toBe(1);
    expect(result.fieldLabel).toBe('船公司');
    expect(onExpandSection).toHaveBeenCalledWith('transport');
    expect(notify).toHaveBeenCalledWith(expect.stringContaining('船公司'));

    window.scrollTo = originalScrollTo;
    document.body.removeChild(container);
  });

  it('findFieldDomElement 在 Form.List 多品目中优先匹配完整路径，不误命中第 0 项', () => {
    const container = document.createElement('div');
    container.innerHTML = `
      <div class="ant-form-item" id="item_0_houseNo">
        <input id="orderForm_documents_0_houseNo" name="documents_0_houseNo" />
      </div>
      <div class="ant-form-item" id="item_1_houseNo">
        <input id="orderForm_documents_1_houseNo" name="documents_1_houseNo" />
      </div>
    `;
    document.body.appendChild(container);

    const el = findFieldDomElement(['documents', 1, 'houseNo'], container);
    expect(el).not.toBeNull();
    document.body.removeChild(container);
  });

  it('scrollToFirstTableError 能够在表格特定行内提取首个错误并横向居中滚动与聚焦', () => {
    const container = document.createElement('div');
    container.innerHTML = `
      <table>
        <tbody>
          <tr data-row-key="fee_row_1">
            <td id="cell_setting">
              <div class="ant-form-item ant-form-item-has-error">
                <div class="ant-form-item-label"><label>费用项目</label></div>
                <div class="ant-form-item-control">
                  <input id="fee_row_1_feeSettingId" name="feeSettingId" />
                  <div class="ant-form-item-explain-error">请选择费用项目</div>
                </div>
              </div>
            </td>
            <td id="cell_party">
              <div class="ant-form-item ant-form-item-has-error">
                <div class="ant-form-item-label"><label>结算单位</label></div>
                <input id="fee_row_1_settlementPartyId" name="settlementPartyId" />
              </div>
            </td>
          </tr>
          <tr data-row-key="fee_row_2">
            <td>
              <div class="ant-form-item ant-form-item-has-error">
                <input id="fee_row_2_feeSettingId" />
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    `;
    document.body.appendChild(container);

    const cellSetting = container.querySelector('#cell_setting') as HTMLElement;
    const scrollIntoViewMock = vi.fn();
    cellSetting.scrollIntoView = scrollIntoViewMock;

    const input = container.querySelector(
      '#fee_row_1_feeSettingId',
    ) as HTMLElement;
    const focusMock = vi.fn();
    input.focus = focusMock;

    const notify = vi.fn();

    const result = scrollToFirstTableError({
      rowKey: 'fee_row_1',
      container,
      notify,
    });

    expect(result.success).toBe(true);
    expect(result.totalErrors).toBe(2);
    expect(result.errorMessage).toBe('请选择费用项目');
    expect(scrollIntoViewMock).toHaveBeenCalledWith({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'center',
    });
    expect(focusMock).toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('请选择费用项目');

    document.body.removeChild(container);
  });

  it('scrollToFirstTableError 在无错误项时安全返回 success: false', () => {
    const container = document.createElement('div');
    container.innerHTML = `<table><tbody><tr data-row-key="ok_row"><td><input /></td></tr></tbody></table>`;
    document.body.appendChild(container);

    const notify = vi.fn();
    const result = scrollToFirstTableError({
      rowKey: 'ok_row',
      container,
      notify,
    });

    expect(result.success).toBe(false);
    expect(result.totalErrors).toBe(0);
    expect(notify).not.toHaveBeenCalled();

    document.body.removeChild(container);
  });
});
