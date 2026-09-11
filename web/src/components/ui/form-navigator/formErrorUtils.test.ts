import {
  collectFormSectionErrors,
  findFieldDomElement,
  findParentSectionKey,
  getFormItemLabel,
  scrollToFirstFormError,
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
    expect(el?.id).toBe('item_1_houseNo');

    document.body.removeChild(container);
  });
});
