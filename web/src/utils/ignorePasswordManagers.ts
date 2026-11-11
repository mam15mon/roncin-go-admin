/** 标记动态挂载的输入框，减少密码管理器把业务输入识别为凭据。 */
export function ignorePasswordManagers(root: HTMLElement) {
  const mark = (element: Element) => {
    if (
      element instanceof HTMLInputElement ||
      element instanceof HTMLTextAreaElement
    ) {
      if (element.getAttribute('autocomplete') !== 'off') {
        element.setAttribute('autocomplete', 'off');
      }
      element.setAttribute('data-1p-ignore', 'true');
      element.setAttribute('data-lpignore', 'true');
      element.setAttribute('data-bwignore', 'true');
      element.setAttribute('data-form-type', 'other');
    }
  };
  const markTree = (element: Element) => {
    mark(element);
    element.querySelectorAll('input, textarea').forEach(mark);
  };
  markTree(root);
  const observer = new MutationObserver((records) => {
    for (const record of records) {
      if (record.type === 'attributes' && record.target instanceof Element) {
        mark(record.target);
      }
      for (const node of record.addedNodes) {
        if (node instanceof Element) markTree(node);
      }
    }
  });
  observer.observe(root, {
    attributes: true,
    attributeFilter: ['autocomplete'],
    childList: true,
    subtree: true,
  });
  return () => observer.disconnect();
}
