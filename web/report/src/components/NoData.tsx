import BeeSad from '../bee_sad.png'


interface NoDataProps {
  message: string
}

function NoData({ message }: Readonly<NoDataProps>) {
  return (
    <div className="no-data">
      <img src={BeeSad} alt="No data" className="no-data-image" />
      <p className="no-data-message">{message}</p>
    </div>
  )
}

export default NoData
