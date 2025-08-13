export const isValidDate = (dateString: string): boolean => {
    const date = new Date(dateString);
    return !isNaN(date.getTime()) && dateString === date.toISOString().split('T')[0];
};