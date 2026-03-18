/**
 * Sums the passed percentage to the R, G or B of a HEX color
 * @param {string} color The color to change
 * @param {number} amount The amount to change the color by
 * @returns {string} The processed part of the color
 */
export function addLight(color: string, amount: number): string {
    const cc = parseInt(color, 16) + amount
    const c = cc > 255 ? 255 : cc
    return c.toString(16).length > 1 ? c.toString(16) : `0${c.toString(16)}`
}

/**
 * Lightens a 6 char HEX color according to the passed percentage
 * @param {string} color The color to change
 * @param {number} amount The amount to change the color by
 * @returns {string} The processed color represented as HEX
 */
export function lighten(color: string, amount: number): string {
    color = color.indexOf('#') >= 0 ? color.substring(1, color.length) : color
    amount = Math.trunc((255 * amount) / 100)
    return `#${addLight(color.substring(0, 2), amount)}${addLight(
        color.substring(2, 4),
        amount
    )}${addLight(color.substring(4, 6), amount)}`
}

/**
 * 绑定实例方法
 * @param instance 实例
 */
export function bindMethods<T extends object>(instance: T): void {
    const prototype = Object.getPrototypeOf(instance);
    const propertyNames = Object.getOwnPropertyNames(prototype);

    propertyNames.forEach((propertyName) => {
        const descriptor = Object.getOwnPropertyDescriptor(prototype, propertyName);
        const propertyValue = instance[propertyName as keyof T];

        if (
            typeof propertyValue === 'function' &&
            propertyName !== 'constructor' &&
            descriptor &&
            !descriptor.get &&
            !descriptor.set
        ) {
            instance[propertyName as keyof T] = propertyValue.bind(instance);
        }
    });
}

/**
 * 滚动到顶部
 */
export function scrollToTop() {
    window.scrollTo({top: 0, behavior: 'smooth'})
}

/**
 * 滚动到底部
 */
export function scrollToBottom() {
    window.scrollTo({top: document.body.scrollHeight, behavior: 'smooth'})
}

/**
 * 滚动到指定元素
 * @param element 目标元素
 */
export function scrollTo(element: HTMLElement) {
    element.scrollIntoView({behavior: 'smooth'})
}
