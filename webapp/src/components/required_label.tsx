// The star is visual only, so the field keeps its plain accessible name.
const requiredLabel = (text: string) => (
    <>
        {text}
        <span
            className='legal-hold-required-star'
            aria-hidden={true}
        >
            {' *'}
        </span>
    </>
);

export default requiredLabel;
